package ml

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"github.com/jakopako/goskyr/scraper"
	"github.com/jakopako/goskyr/utils"
	"github.com/sjwhitworth/golearn/base"
	"github.com/sjwhitworth/golearn/evaluation"
	"github.com/sjwhitworth/golearn/knn"
)

//////////////////////
// Feature Extraction
//////////////////////

// Features contains all the relevant features and the class label
type Features struct {
	letterFrequencies []int
	digitCount        int
	runeCount         int
	dictWordsCount    int
	slashCount        int
	colonCount        int
	dashCount         int
	dotCount          int
	whitespaceCount   int
	class             string
}

// NonAlphaFeatureList contains a list of strings representing the Features excluding the letter frequencies
var NonAlphaFeatureList []string = []string{
	"digit-count",
	"rune-count",
	"dict-words-count",
	"slash-count",
	"colon-count",
	"dash-count",
	"dot-count",
	"whitespace-count",
	"class",
}

// ExtractFeatures extracts features based on a given configuration and a directory
// containing words of different languages. Those features can then be used to train
// a ML model to automatically classify scraped fields for new websites.
func ExtractFeatures(config *scraper.Config, featureFile, wordsDir string) error {
	var calcWg sync.WaitGroup
	var writerWg sync.WaitGroup
	wordMap, err := loadWords(wordsDir)
	if err != nil {
		return err
	}
	featuresChan := make(chan *Features)
	writerWg.Add(1)
	go writeFeaturesToFile(featureFile, featuresChan, &writerWg)
	for _, s := range config.Scrapers {
		calcWg.Add(1)
		go calculateScraperFeatures(s, featuresChan, wordMap, &config.Global, &calcWg)
	}
	calcWg.Wait()
	close(featuresChan)
	writerWg.Wait()

	return nil
}

func loadWords(wordsDir string) (map[string]bool, error) {
	words := map[string]bool{}
	return words, filepath.WalkDir(wordsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			scanner := bufio.NewScanner(file)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				words[strings.ToLower(scanner.Text())] = true
			}
			file.Close()
		}
		return nil
	})
}

func writeFeaturesToFile(filename string, featuresChan <-chan *Features, wg *sync.WaitGroup) {
	defer wg.Done()
	featuresFile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	if err := featuresFile.Truncate(0); err != nil {
		log.Fatal(err)
	}
	_, err = featuresFile.Seek(0, 0)
	if err != nil {
		log.Fatal(err)
	}
	defer featuresFile.Close()
	writer := bufio.NewWriter(featuresFile)
	alphabet := []string{}
	for r := 'a'; r <= 'z'; r++ {
		alphabet = append(alphabet, string(r))
	}
	writer.WriteString(fmt.Sprintf("%s, %s\n", strings.Join(alphabet, ", "), strings.Join(NonAlphaFeatureList, ", ")))
	for f := range featuresChan {
		lfStrings := []string{}
		for _, lf := range f.letterFrequencies {
			lfStrings = append(lfStrings, fmt.Sprintf("%d", lf))
		}

		writer.WriteString(fmt.Sprintf("%s, %d, %d, %d, %d, %d, %d, %d, %d, %s\n",
			strings.Join(lfStrings, ", "),
			f.digitCount,
			f.runeCount,
			f.dictWordsCount,
			f.slashCount,
			f.colonCount,
			f.dashCount,
			f.dotCount,
			f.whitespaceCount,
			f.class))
	}
	writer.Flush()
}

func calculateScraperFeatures(s scraper.Scraper, featuresChan chan<- *Features, wordMap map[string]bool, globalConfig *scraper.GlobalConfig, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("calculating features for %s\n", s.Name)
	result, err := s.Scrape(globalConfig, true)
	if err != nil {
		log.Printf("%s ERROR: %s", s.Name, err)
		return
	}
	for _, item := range result.Items {
		for fName, fValue := range item {
			fValueString := fValue.(string)
			f := calculateFeatures(fName, fValueString, wordMap)
			featuresChan <- &f
		}
	}
}

func calculateFeatures(fName, fValue string, wordMap map[string]bool) Features {
	return Features{
		letterFrequencies: countLetterFrequencies(fValue),
		digitCount:        countDigits(fValue),
		runeCount:         countRunes(fValue),
		dictWordsCount:    countDictWords(fValue, wordMap),
		slashCount:        countRune(fValue, []rune("/")[0]),
		colonCount:        countRune(fValue, []rune(":")[0]),
		dashCount:         countRune(fValue, []rune("-")[0]),
		dotCount:          countRune(fValue, []rune(".")[0]),
		whitespaceCount:   countRune(fValue, []rune(" ")[0]),
		class:             fName,
	}
}

func countLetterFrequencies(s string) []int {
	fs := make([]int, 26)
	for _, r := range s {
		rl := unicode.ToLower(r)
		if rl >= 'a' && rl <= 'z' {
			fs[int(rl)-97]++
		}
	}
	return fs
}

func countDigits(s string) int {
	c := 0
	for _, r := range s {
		if unicode.IsDigit(r) {
			c++
		}
	}
	return c
}

func countRunes(s string) int {
	return len(s)
}

func countDictWords(s string, wordMap map[string]bool) int {
	c := 0
	words := strings.Split(strings.ToLower(s), " ")
	for _, w := range words {
		if _, found := wordMap[w]; found {
			c++
		}
	}
	return c
}

func countRune(s string, r rune) int {
	c := 0
	for _, l := range s {
		if l == r {
			c++
		}
	}
	return c
}

///////////////////////
// ML model generation
///////////////////////

func TrainModel(filename string) error {
	log.Printf("loading csv data from file %s\n", filename)
	rawData, err := base.ParseCSVToInstances(filename, true)
	if err != nil {
		return err
	}
	log.Println("initializing KNN classifier")
	cls := knn.NewKnnClassifier("euclidean", "linear", 2)
	log.Println("performing a training-test split")
	trainData, testData := base.InstancesTrainTestSplit(rawData, 0.75)
	log.Println("training on trainData")
	cls.Fit(trainData)
	predictions, err := cls.Predict(testData)
	if err != nil {
		return err
	}
	confusionMat, err := evaluation.GetConfusionMatrix(testData, predictions)
	if err != nil {
		return err
	}
	fmt.Println(evaluation.GetSummary(confusionMat))
	modelFilename := "goskyr.model"
	classesFileName := "goskyr.class"
	log.Printf("storing model to files %s and %s\n", modelFilename, classesFileName)
	if err := cls.Save(modelFilename); err != nil { // no idea why cls.Save prints this line 'writer: ...'
		return err
	}
	classValues := trainData.AllClassAttributes()[0].(*base.CategoricalAttribute).GetValues()
	f, err := os.Create(classesFileName)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, value := range classValues {
		fmt.Fprintln(f, value)
	}
	return nil
}

//////////////////
// Label new data
//////////////////

type Labler struct {
	wordMap   map[string]bool
	cls       *knn.KNNClassifier
	classAttr *base.CategoricalAttribute
}

func LoadLabler(modelName, wordListsDir string) (*Labler, error) {
	w, err := loadWords(wordListsDir)
	if err != nil {
		return nil, err
	}
	cls := knn.NewKnnClassifier("euclidean", "linear", 2)
	modelFname := fmt.Sprintf("%s.model", modelName)
	if err := cls.Load(modelFname); err != nil {
		return nil, err
	}
	classAttr := new(base.CategoricalAttribute)
	classAttr.SetName("class")
	classFname := fmt.Sprintf("%s.class", modelName)
	file, err := os.Open(classFname)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		classAttr.GetSysValFromString(scanner.Text())
	}

	ll := &Labler{
		wordMap:   w,
		cls:       cls,
		classAttr: classAttr,
	}
	return ll, nil
}

func (ll *Labler) PredictLabel(fValue ...string) (string, error) {
	featVectors := []*Features{}
	for _, v := range fValue {
		f := calculateFeatures("", v, ll.wordMap)
		featVectors = append(featVectors, &f)
	}
	attrs := make([]base.Attribute, len(NonAlphaFeatureList)+26)
	for r := 'a'; r <= 'z'; r++ {
		attrs[int(r)-97] = base.NewFloatAttribute(string(r))
	}
	for i := 26; i < len(attrs)-1; i++ {
		attrs[i] = base.NewFloatAttribute(NonAlphaFeatureList[i-26])
	}
	attrs[len(attrs)-1] = ll.classAttr

	predictions := []string{}
	for _, f := range featVectors {
		newInst := base.NewDenseInstances()
		newSpecs := make([]base.AttributeSpec, len(attrs))
		for i, a := range attrs {
			newSpecs[i] = newInst.AddAttribute(a)
		}
		newInst.Extend(1)

		newInst.AddClassAttribute(newInst.AllAttributes()[len(attrs)-1])

		for i := 0; i < 26; i++ {
			newInst.Set(newSpecs[i], 0, newSpecs[i].GetAttribute().GetSysValFromString(fmt.Sprint(f.letterFrequencies[i])))
		}
		newInst.Set(newSpecs[26], 0, newSpecs[26].GetAttribute().GetSysValFromString(fmt.Sprint(f.digitCount)))
		newInst.Set(newSpecs[27], 0, newSpecs[27].GetAttribute().GetSysValFromString(fmt.Sprint(f.runeCount)))
		newInst.Set(newSpecs[28], 0, newSpecs[28].GetAttribute().GetSysValFromString(fmt.Sprint(f.dictWordsCount)))
		newInst.Set(newSpecs[29], 0, newSpecs[29].GetAttribute().GetSysValFromString(fmt.Sprint(f.slashCount)))
		newInst.Set(newSpecs[30], 0, newSpecs[30].GetAttribute().GetSysValFromString(fmt.Sprint(f.colonCount)))
		newInst.Set(newSpecs[31], 0, newSpecs[31].GetAttribute().GetSysValFromString(fmt.Sprint(f.dashCount)))
		newInst.Set(newSpecs[32], 0, newSpecs[32].GetAttribute().GetSysValFromString(fmt.Sprint(f.dotCount)))
		newInst.Set(newSpecs[33], 0, newSpecs[33].GetAttribute().GetSysValFromString(fmt.Sprint(f.whitespaceCount)))
		pred, err := ll.cls.Predict(newInst)
		if err != nil {
			return "", err
		}
		predictions = append(predictions, pred.RowString(0))
	}
	pred := utils.MostOcc(predictions)
	return pred, nil
}
