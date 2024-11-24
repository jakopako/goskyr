package main

import (
	"flag"
	"fmt"
	"log/slog"
	"math"
	"os"
	"runtime/debug"
	"sort"
	"strconv"
	"sync"

	"github.com/jakopako/goskyr/autoconfig"
	"github.com/jakopako/goskyr/config"
	"github.com/jakopako/goskyr/ml"
	"github.com/jakopako/goskyr/output"
	"github.com/jakopako/goskyr/scraper"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v3"
)

var version = "dev"

func worker(sc <-chan scraper.Scraper, ic chan<- map[string]interface{}, stc chan<- scraper.ScrapingStats, gc *scraper.GlobalConfig, threadNr int) {
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
		stc <- *result.Stats
	}
	workerLogger.Info("done working")
}

func collectAllStats(stc <-chan scraper.ScrapingStats) []scraper.ScrapingStats {
	result := []scraper.ScrapingStats{}
	for st := range stc {
		result = append(result, st)
	}
	return result
}

func printAllStats(stats []scraper.ScrapingStats) {
	slog.Info("printing scraper summary")
	// sort by name alphabetically
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Name < stats[j].Name
	})

	total := scraper.ScrapingStats{
		Name: "total",
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Items", "Errors"})

	for _, s := range stats {
		row := []string{s.Name, strconv.Itoa(s.NrItems), strconv.Itoa(s.NrErrors)}
		if s.NrErrors > 0 {
			table.Rich(row, []tablewriter.Colors{{tablewriter.Normal, tablewriter.FgRedColor}, {tablewriter.Normal, tablewriter.FgRedColor}, {tablewriter.Normal, tablewriter.FgRedColor}})
		} else if s.NrErrors == 0 && s.NrItems == 0 {
			table.Rich(row, []tablewriter.Colors{{tablewriter.Normal, tablewriter.FgYellowColor}, {tablewriter.Normal, tablewriter.FgYellowColor}, {tablewriter.Normal, tablewriter.FgYellowColor}})
		} else {
			table.Append(row)
		}
		total.NrErrors += s.NrErrors
		total.NrItems += s.NrItems
	}
	table.SetFooter([]string{total.Name, strconv.Itoa(total.NrItems), strconv.Itoa(total.NrErrors)})
	table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT})
	table.SetBorder(false)
	table.Render()
}

func main() {
	singleScraper := flag.String("s", "", "The name of the scraper to be run.")
	toStdout := flag.Bool("stdout", false, "If set to true the scraped data will be written to stdout despite any other existing writer configurations. In combination with the -generate flag the newly generated config will be written to stdout instead of to a file.")
	configLoc := flag.String("c", "./config.yml", "The location of the configuration. Can be a directory containing config files or a single config file.")
	printVersion := flag.Bool("v", false, "The version of goskyr.")
	generateConfig := flag.String("g", "", "Automatically generate a config file for the given url.")
	m := flag.Int("m", 20, "The minimum number of items on a page. This is needed to filter out noise. Works in combination with the -g flag.")
	f := flag.Bool("f", false, "Only show fields that have varying values across the list of items. Works in combination with the -g flag.")
	renderJs := flag.Bool("r", false, "Render JS before generating a configuration file. Works in combination with the -g flag.")
	extractFeatures := flag.String("e", "", "Extract ML features based on the given configuration file (-c) and write them to the given file in csv format.")
	wordsDir := flag.String("w", "word-lists", "The directory that contains a number of files containing words of different languages. This is needed for the ML part (use with -e or -b).")
	buildModel := flag.String("t", "", "Train a ML model based on the given csv features file. This will generate 2 files, goskyr.model and goskyr.class")
	modelPath := flag.String("model", "", "Use a pre-trained ML model to infer names of extracted fields. Works in combination with the -g flag.")
	debugFlag := flag.Bool("debug", false, "Prints debug logs and writes scraped html's to files.")
	summaryFlag := flag.Bool("summary", false, "Print scraper summary at the end.")

	flag.Parse()

	if *printVersion {
		buildInfo, ok := debug.ReadBuildInfo()
		if ok {
			if buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
				fmt.Println(buildInfo.Main.Version)
				return
			}
		}
		fmt.Println(version)
		return
	}

	config.Debug = *debugFlag
	var logLevel slog.Level
	if *debugFlag {
		logLevel = slog.LevelDebug
	} else {
		logLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	if *generateConfig != "" {
		slog.Debug("starting to generate config")
		s := &scraper.Scraper{URL: *generateConfig}
		if *renderJs {
			s.RenderJs = true
		}
		slog.Debug(fmt.Sprintf("analyzing url %s", s.URL))
		err := autoconfig.GetDynamicFieldsConfig(s, *m, *f, *modelPath, *wordsDir)
		if err != nil {
			slog.Error(fmt.Sprintf("%v", err))
			os.Exit(1)
		}
		c := scraper.Config{
			Scrapers: []scraper.Scraper{
				*s,
			},
		}
		yamlData, err := yaml.Marshal(&c)
		if err != nil {
			slog.Error(fmt.Sprintf("error while marshaling. %v", err))
			os.Exit(1)
		}

		if *toStdout {
			fmt.Println(string(yamlData))
		} else {
			f, err := os.Create(*configLoc)
			if err != nil {
				slog.Error(fmt.Sprintf("error opening file: %v", err))
				os.Exit(1)
			}
			defer f.Close()
			_, err = f.Write(yamlData)
			if err != nil {
				slog.Error(fmt.Sprintf("error writing to file: %v", err))
				os.Exit(1)
			}
			slog.Info(fmt.Sprintf("successfully wrote config to file %s", *configLoc))
		}
		return
	}

	if *buildModel != "" {
		if err := ml.TrainModel(*buildModel); err != nil {
			slog.Error(fmt.Sprintf("%v", err))
			os.Exit(1)
		}
		return
	}

	config, err := scraper.NewConfig(*configLoc)
	if err != nil {
		slog.Error(fmt.Sprintf("%v", err))
		os.Exit(1)
	}

	if *extractFeatures != "" {
		if err := ml.ExtractFeatures(config, *extractFeatures, *wordsDir); err != nil {
			slog.Error(fmt.Sprintf("%v", err))
			os.Exit(1)
		}
		return
	}

	var workerWg sync.WaitGroup
	var writerWg sync.WaitGroup
	var statsWg sync.WaitGroup
	ic := make(chan map[string]interface{})

	var writer output.Writer
	if *toStdout {
		writer = &output.StdoutWriter{}
	} else {
		switch config.Writer.Type {
		case output.STDOUT_WRITER_TYPE:
			writer = &output.StdoutWriter{}
		case output.API_WRITER_TYPE:
			writer = output.NewAPIWriter(&config.Writer)
		case output.FILE_WRITER_TYPE:
			writer = output.NewFileWriter(&config.Writer)
		default:
			slog.Error(fmt.Sprintf("writer of type %s not implemented", config.Writer.Type))
			os.Exit(1)
		}
	}

	if config.Global.UserAgent == "" {
		config.Global.UserAgent = "goskyr web scraper (github.com/jakopako/goskyr)"
	}

	sc := make(chan scraper.Scraper)
	stc := make(chan scraper.ScrapingStats)

	// fill worker queue
	go func() {
		for _, s := range config.Scrapers {
			if *singleScraper == "" || *singleScraper == s.Name {
				// s.Debug = *debugFlag
				sc <- s
			}
		}
		close(sc)
	}()

	// start workers
	nrWorkers := 1
	if *singleScraper == "" {
		nrWorkers = int(math.Min(20, float64(len(config.Scrapers))))
	}
	slog.Info(fmt.Sprintf("running with %d threads", nrWorkers))
	workerWg.Add(nrWorkers)
	slog.Debug("starting workers")
	for i := 0; i < nrWorkers; i++ {
		go func(j int) {
			defer workerWg.Done()
			worker(sc, ic, stc, &config.Global, j)
		}(i)
	}

	// start writer
	writerWg.Add(1)
	slog.Debug("starting writer")
	go func() {
		defer writerWg.Done()
		writer.Write(ic)
	}()

	// start stats collection
	statsWg.Add(1)
	slog.Debug("starting stats collection")
	go func() {
		defer statsWg.Done()
		allStats := collectAllStats(stc)
		writerWg.Wait() // only print stats in the end
		// a bit ugly to just check here and do the collection
		// of the stats even though they might not be needed.
		// But this is easier for now, coding-wise.
		if *summaryFlag {
			printAllStats(allStats)
		}
	}()

	workerWg.Wait()
	close(ic)
	close(stc)
	writerWg.Wait()
	statsWg.Wait()
}
