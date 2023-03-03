package ml

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"unicode"

	"github.com/jakopako/goskyr/scraper"
	"github.com/sjwhitworth/golearn/base"
	"github.com/sjwhitworth/golearn/evaluation"
	"github.com/sjwhitworth/golearn/knn"
)

type Labler struct {
	wordMap map[string]bool
	cls     *knn.KNNClassifier
}

func LoadLabler() (*Labler, error) {
	w, err := loadWords()
	if err != nil {
		return nil, err
	}
	cls := knn.NewKnnClassifier("euclidean", "linear", 2)
	cls.Load("croncert.model")
	ll := &Labler{
		wordMap: w,
		cls:     cls,
	}
	return ll, nil
}

func (ll *Labler) PredictLabel(fValue ...string) string {
	features := []*Features{}
	for _, v := range fValue {
		f := calculateFeatures("", v, ll.wordMap)
		// f.class = "title"
		features = append(features, &f)
	}
	// https://github.com/sjwhitworth/golearn/blob/master/examples/instances/instances.go
	attrs := make([]base.Attribute, 8)
	for i := 0; i < 7; i++ {
		attrs[i] = base.NewFloatAttribute(FeatureList[i])
	}
	attrs[7] = new(base.CategoricalAttribute)
	attrs[7].SetName(FeatureList[7])
	for _, cl := range Classes {
		attrs[7].GetSysValFromString(cl)
	}

	newInst := base.NewDenseInstances()
	newSpecs := make([]base.AttributeSpec, len(attrs))
	for i, a := range attrs {
		newSpecs[i] = newInst.AddAttribute(a)
	}
	newInst.Extend(1)

	// fmt.Println(newInst.AllAttributes())

	newInst.AddClassAttribute(newInst.AllAttributes()[7])

	newInst.Set(newSpecs[0], 0, newSpecs[0].GetAttribute().GetSysValFromString(fmt.Sprint(features[0].digitCount)))
	newInst.Set(newSpecs[1], 0, newSpecs[1].GetAttribute().GetSysValFromString(fmt.Sprint(features[0].runeCount)))
	newInst.Set(newSpecs[2], 0, newSpecs[2].GetAttribute().GetSysValFromString(fmt.Sprint(features[0].dictWordsCount)))
	newInst.Set(newSpecs[3], 0, newSpecs[3].GetAttribute().GetSysValFromString(fmt.Sprint(features[0].slashCount)))
	newInst.Set(newSpecs[4], 0, newSpecs[4].GetAttribute().GetSysValFromString(fmt.Sprint(features[0].colonCount)))
	newInst.Set(newSpecs[5], 0, newSpecs[5].GetAttribute().GetSysValFromString(fmt.Sprint(features[0].dashCount)))
	newInst.Set(newSpecs[6], 0, newSpecs[6].GetAttribute().GetSysValFromString(fmt.Sprint(features[0].dotCount)))
	// newInst.Set(newSpecs[7], 0, newSpecs[7].GetAttribute().GetSysValFromString(fmt.Sprint(features[0].class)))
	fmt.Println(newInst)
	pred, _ := ll.cls.Predict(newInst)
	// fmt.Println(err)
	// fmt.Println(pred)
	fmt.Printf("\n\nPrediction: %s\n", pred.RowString(0))
	return pred.RowString(0)
}

// Features contains all the relevant features and the class label
type Features struct {
	digitCount     int
	runeCount      int
	dictWordsCount int
	slashCount     int
	colonCount     int
	dashCount      int
	dotCount       int
	class          string
}

var FeatureList []string = []string{
	"digit-count",
	"rune-count",
	"dict-words-count",
	"slash-count",
	"colon-count",
	"dash-count",
	"dot-count",
	"class",
}

var Classes []string = []string{
	"date-component-time",
	"title",
	"url",
	"date-component-day",
	"date-component-month",
	"comment",
	"date-component-day-month",
	"date-component-day-month-year-time",
	"date-component-day-month-year",
	"date-component-year",
	"date-component-day-month-time",
	"date-component-month-year"}

func calculateFeatures(fName, fValue string, wordMap map[string]bool) Features {
	return Features{
		digitCount:     countDigits(fValue),
		runeCount:      countRunes(fValue),
		dictWordsCount: countDictWords(fValue, wordMap),
		slashCount:     countRune(fValue, []rune("/")[0]),
		colonCount:     countRune(fValue, []rune(":")[0]),
		dashCount:      countRune(fValue, []rune("-")[0]),
		dotCount:       countRune(fValue, []rune(".")[0]),
		class:          fName,
	}
}

func calculateScraperFeatures(s scraper.Scraper, featuresChan chan<- *Features, wordMap map[string]bool, globalConfig *scraper.GlobalConfig, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("calculating features for %s\n", s.Name)
	items, err := s.GetItems(globalConfig, true)
	if err != nil {
		log.Printf("%s ERROR: %s", s.Name, err)
		return
	}
	for _, item := range items {
		for fName, fValue := range item {
			fValueString := fValue.(string)
			f := calculateFeatures(fName, fValueString, wordMap)
			featuresChan <- &f
			// fmt.Printf("%+v\n", f)
			// fmt.Println(fValueString)
		}
	}
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
	writer.WriteString("digit-count, rune-count, dict-words-count, slash-count, colon-count, dash-count, dot-count, class\n")
	for f := range featuresChan {
		// fmt.Println(f)
		writer.WriteString(fmt.Sprintf("%d, %d, %d, %d, %d, %d, %d, %s\n",
			f.digitCount,
			f.runeCount,
			f.dictWordsCount,
			f.slashCount,
			f.colonCount,
			f.dashCount,
			f.dotCount,
			f.class))
	}
	writer.Flush()
}

func ExtractFeatures(config *scraper.Config, filename string) error {
	var calcWg sync.WaitGroup
	var writerWg sync.WaitGroup
	wordMap, err := loadWords()
	if err != nil {
		return err
	}
	featuresChan := make(chan *Features)
	writerWg.Add(1)
	go writeFeaturesToFile(filename, featuresChan, &writerWg)
	for _, s := range config.Scrapers {
		calcWg.Add(1)
		go calculateScraperFeatures(s, featuresChan, wordMap, &config.Global, &calcWg)
	}
	calcWg.Wait()
	close(featuresChan)
	writerWg.Wait()

	// fmt.Println(len(features))
	// for _, f := range features {
	// 	fmt.Println(f)
	// }

	return nil
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

func BuildModel(filename string) error {
	log.Printf("loading csv data from file %s\n", filename)
	rawData, err := base.ParseCSVToInstances(filename, true)
	if err != nil {
		return err
	}
	fmt.Println(rawData)
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
	// fmt.Println(predictions)
	fmt.Println(evaluation.GetSummary(confusionMat))
	modelFName := "croncert.model"
	log.Printf("storing model to file %s\n", modelFName)
	return cls.Save(modelFName)
}

func loadWords() (map[string]bool, error) {
	words := map[string]bool{}
	for _, fn := range []string{
		"word-lists/english.dic",
		"word-lists/francais.txt",
		"word-lists/wordlist-german.txt",
	} {
		file, err := os.Open(fn)
		if err != nil {
			return words, err
		}
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			words[strings.ToLower(scanner.Text())] = true
		}
		file.Close()
	}
	return words, nil
}
