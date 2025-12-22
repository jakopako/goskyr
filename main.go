/*
goskyr is a command line web scraper written in Go.

Have a look at the README.md for more information.
*/
package main

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"runtime/debug"
	"slices"
	"strings"
	"sync"

	"github.com/alecthomas/kong"
	"github.com/jakopako/goskyr/autoconfig"
	"github.com/jakopako/goskyr/config"
	"github.com/jakopako/goskyr/fetch"
	"github.com/jakopako/goskyr/ml"
	"github.com/jakopako/goskyr/output"
	"github.com/jakopako/goskyr/scraper"
	"github.com/jakopako/goskyr/types"
	"github.com/miekg/king"
	"gopkg.in/yaml.v3"
)

var version = "dev"

const name = "goskyr"

type VersionFlag string

func (v VersionFlag) Decode(_ *kong.DecodeContext) error { return nil }
func (v VersionFlag) IsBool() bool                       { return true }
func (v VersionFlag) BeforeApply(app *kong.Kong, vars kong.Vars) error {
	fmt.Println(vars["version"])
	app.Exit(0)
	return nil
}

type cli struct {
	Version VersionFlag `short:"v" long:"version" help:"Print the version and exit."`
	Debug   bool        `short:"d" long:"debug" help:"Set log level to 'debug' and store additional helpful debugging data."`

	Completion CompletionCommand `cmd:"" help:"Generate autocompletion file."`

	Scrape   ScrapeCmd   `cmd:"" help:"Scrape data"`
	Generate GenerateCmd `cmd:"" help:"Generate a scraper configuration file for the given URL"`
	Extract  ExtractCmd  `cmd:"" help:"Extract ML features based on the given configuration file"`
	Train    TrainCmd    `cmd:"" help:"Train ML model based on the given features file. This will generate 2 files, goskyr.model and goskyr.class"`
	List     ListCmd     `cmd:"" help:"List available scrapers in the given configuration file(s)"`
}

type ShellType string

const (
	BASH ShellType = "bash"
	ZSH  ShellType = "zsh"
	FISH ShellType = "fish"
)

var shellTypes = []string{string(BASH), string(ZSH), string(FISH)}

type CompletionCommand struct {
	Shell ShellType `short:"s" help:"The shell that you want to create the autocompletion file for." required:""`
}

func (acc *CompletionCommand) Run() error {
	cli := &cli{}
	parser := kong.Must(cli)

	switch acc.Shell {
	case BASH:
		b := &king.Bash{}
		b.Completion(parser.Model.Node, name)
		return b.Write()
	case ZSH:
		z := &king.Zsh{}
		z.Completion(parser.Model.Node, name)
		return z.Write()
	case FISH:
		f := &king.Fish{}
		f.Completion(parser.Model.Node, name)
		return f.Write()
	default:
		return fmt.Errorf("shell type not supported: %s. Must be one of [%s].", acc.Shell, strings.Join(shellTypes, ", "))
	}
}

type ScrapeCmd struct {
	Config string `short:"c" default:"./config.yml" help:"The location of the configuration. Can be a directory containing config files or a single config file." completion:"<file>"`
	Name   string `short:"n" help:"The name of the scraper to be run, if only one of the configured ones should be run."`
	Stdout bool   `short:"o" help:"If set to true the scraped data will be written to stdout despite any other existing writer configurations."`
	DryRun bool   `short:"D" help:"If set to true the scraper will not persist any scraped data (currently only has an effect on the APIWriter)."`
}

func (scc *ScrapeCmd) Run() error {
	config, err := scraper.NewConfig(scc.Config)
	if err != nil {
		slog.Error(fmt.Sprintf("%v", err))
		return err
	}

	if scc.Stdout {
		config.Writer.Type = output.STDOUT_WRITER_TYPE
	}

	if scc.DryRun {
		config.Writer.DryRun = true
	}

	writer, err := output.NewWriter(&config.Writer)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	scraperChan := make(chan scraper.Scraper)
	var statusChan chan types.ScraperStatus = nil
	if config.Writer.WriteStatus && !config.Writer.DryRun {
		slog.Info("scraper status collection enabled")
		statusChan = make(chan types.ScraperStatus)
	} else {
		slog.Info("scraper status collection disabled")
	}

	// fill worker queue
	go func() {
		if scc.Name == "" {
			slog.Info(fmt.Sprintf("queueing %d scrapers", len(config.Scrapers)))
			for _, s := range config.Scrapers {
				scraperChan <- s
			}
		} else {
			foundSingleScraper := false
			for _, s := range config.Scrapers {
				if scc.Name == s.Name {
					scraperChan <- s
					foundSingleScraper = true
					break
				}
			}
			if !foundSingleScraper {
				slog.Error(fmt.Sprintf("no scrapers found for name %s", scc.Name))
				os.Exit(1)
			}
		}
		close(scraperChan)
	}()

	// start workers
	nrWorkers := 1
	if scc.Name == "" {
		nrWorkers = int(math.Min(20, float64(len(config.Scrapers))))
	}
	slog.Info(fmt.Sprintf("running with %d threads", nrWorkers))

	workerWg := sync.WaitGroup{}
	workerWg.Add(nrWorkers)

	itemChan := make(chan map[string]any)
	slog.Debug("starting workers")
	for i := range nrWorkers {
		go func(j int) {
			defer workerWg.Done()
			worker(scraperChan, itemChan, statusChan, config.Global, j)
		}(i)
	}

	// start collector (collecting items and possibly scraper status)
	collectorWg := sync.WaitGroup{}
	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		slog.Debug("starting collector")
		collector(itemChan, statusChan, writer)
	}()

	workerWg.Wait()
	slog.Debug("all workers finished, closing item channel")
	close(itemChan)
	if statusChan != nil {
		slog.Debug("all workers finished, closing scraper status channel")
		close(statusChan)
	}
	collectorWg.Wait()
	return nil
}

func worker(sc <-chan scraper.Scraper, ic chan<- map[string]any, stc chan<- types.ScraperStatus, gc *scraper.GlobalConfig, threadNr int) {
	workerLogger := slog.With(slog.Int("thread", threadNr))
	for s := range sc {
		scraperLogger := workerLogger.With(slog.String("name", s.Name))
		scraperLogger.Info("starting scraping task")
		result, err := s.Scrape(gc, false)
		if err != nil {
			scraperLogger.Error(fmt.Sprintf("%s: %s", s.Name, err))
			continue
		}
		scraperLogger.Info(fmt.Sprintf("fetched %d items", result.Stats.NrItems))
		for _, item := range result.Items {
			ic <- item
		}
		// if the scraper status channel is not nil, it means that we are collecting stats
		if stc != nil {
			stc <- *result.Stats
		}
	}
	workerLogger.Info("done working")
}

func collector(itemChan <-chan map[string]any, statusChan <-chan types.ScraperStatus, writer output.Writer) {
	collectorLogger := slog.With(slog.String("collector", "main"))
	writerWg := sync.WaitGroup{}
	writerWg.Add(1)
	go func() {
		defer writerWg.Done()
		collectorLogger.Debug("starting writing items")
		writer.Write(itemChan)
	}()

	if statusChan != nil {
		statusWg := sync.WaitGroup{}
		statusWg.Add(1)
		go func() {
			defer statusWg.Done()
			collectorLogger.Debug("starting writing scraper status")
			writer.WriteStatus(statusChan)
		}()
		statusWg.Wait()
		collectorLogger.Debug("done writing scraper status")
	}
	writerWg.Wait()
	collectorLogger.Debug("done writing items")
}

type GenerateCmd struct {
	URL           string `short:"u" long:"url" help:"The URL for which to generate the scraper configuration file." required:""`
	MinOccurrence int    `short:"m" default:"20" help:"The minimum number of occurrences of a certain field on an html page to be included in the suggested fields. This is needed to filter out noise."`
	Distinct      bool   `short:"D" help:"If set to true only fields with distinct values will be included in the suggested fields."`
	RenderJS      bool   `short:"r" help:"Render javascript before analyzing the html page."`
	WordLists     string `short:"w" default:"word-lists" help:"The directory that contains a number of files containing words of different languages, needed for extracting ML features." completion:"<directory>"`
	ModelName     string `short:"M" help:"The name to a pre-trained ML model to infer names of extracted fields." completion:"<file>"`
	Stdout        bool   `short:"o" long:"stdout" help:"If set to true the the generated configuration will be written to stdout."`
	Config        string `short:"c" long:"config" default:"./config.yml" help:"The file that the generated configuration will be written to." completion:"<file>"`
}

func (g *GenerateCmd) Run() error {
	slog.Debug("starting to generate config")
	s := &scraper.Scraper{
		URL: g.URL,
		FetcherConfig: fetch.FetcherConfig{
			Type: fetch.STATIC_FETCHER_TYPE, // default to static fetcher
		},
	}

	if g.RenderJS {
		s.FetcherConfig.Type = fetch.DYNAMIC_FETCHER_TYPE
	}

	slog.Debug(fmt.Sprintf("analyzing url %s", s.URL))
	err := autoconfig.GenerateConfig(s, g.MinOccurrence, g.Distinct, g.ModelName, g.WordLists)
	if err != nil {
		slog.Error(fmt.Sprintf("%v", err))
		return err
	}

	c := scraper.Config{
		Scrapers: []scraper.Scraper{
			*s,
		},
	}
	yamlData, err := yaml.Marshal(&c)
	if err != nil {
		slog.Error(fmt.Sprintf("error while marshalling. %v", err))
		return err
	}

	if g.Stdout {
		fmt.Println(string(yamlData))
	} else {
		f, err := os.Create(g.Config)
		if err != nil {
			slog.Error(fmt.Sprintf("error opening file: %v", err))
			return err
		}
		defer f.Close()

		_, err = f.Write(yamlData)
		if err != nil {
			slog.Error(fmt.Sprintf("error writing to file: %v", err))
			return err
		}
		slog.Info(fmt.Sprintf("successfully wrote config to file %s", g.Config))
	}

	return nil
}

type ExtractCmd struct {
	Config    string `short:"c" default:"./config.yml" help:"The location of the configuration file." completion:"<file>"`
	OutFile   string `short:"o" help:"The file to which the extracted features will be written in csv format." required:""`
	WordLists string `short:"w" default:"word-lists" help:"The directory that contains a number of files containing words of different languages, needed for extracting ML features." completion:"<directory>"`
}

func (e *ExtractCmd) Run() error {
	config, err := scraper.NewConfig(e.Config)
	if err != nil {
		slog.Error(fmt.Sprintf("%v", err))
		return err
	}

	if err := ml.ExtractFeatures(config, e.OutFile, e.WordLists); err != nil {
		slog.Error(fmt.Sprintf("%v", err))
		return err
	}

	return nil
}

type TrainCmd struct {
	FeatureFile string `short:"f" help:"The csv file containing the extracted features." required:""`
}

func (t *TrainCmd) Run() error {
	if err := ml.TrainModel(t.FeatureFile); err != nil {
		slog.Error(fmt.Sprintf("%v", err))
		return err
	}

	slog.Info("successfully trained model")
	return nil
}

type ListCmd struct {
	Config     string `short:"c" default:"./config.yml" help:"The location of the configuration. Can be a directory containing config files or a single config file." completion:"<file>"`
	Completion bool   `short:"C" help:"If set to true, the output will be formatted for autocompletion scripts."`
}

func (lc *ListCmd) Run() error {
	config, err := scraper.NewConfig(lc.Config)
	if err != nil {
		if lc.Completion {
			// in completion mode, we just return an empty output on error
			return nil
		}
		slog.Error(fmt.Sprintf("%v", err))
		return err
	}

	names := make([]string, 0, len(config.Scrapers))
	for _, s := range config.Scrapers {
		names = append(names, s.Name)
	}

	slices.Sort(names)
	for _, name := range names {
		fmt.Println(name)
	}

	return nil
}

func getVersion() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if ok {
		if buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
			// fmt.Println(buildInfo.Main.Version)
			return buildInfo.Main.Version
		}
	}
	// fmt.Println(version)
	return version
}

func initializeLogging(debug bool) {
	var logLevel slog.Level
	if debug {
		logLevel = slog.LevelDebug
	} else {
		logLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)
}

func main() {
	cli := cli{
		Version: VersionFlag(getVersion()),
	}

	ctx := kong.Parse(&cli,
		kong.Vars{
			"version": string(cli.Version),
		})

	initializeLogging(cli.Debug)
	config.Debug = cli.Debug

	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
