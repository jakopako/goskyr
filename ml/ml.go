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

type Features struct {
	digitCount     int    `csv:"digit-count"`
	runeCount      int    `csv:"rune-count"`
	dictWordsCount int    `csv:"dict-words-count"`
	slashCount     int    `csv:"slash-count"`
	colonCount     int    `csv:"colon-count"`
	dashCount      int    `csv:"dash-count"`
	dotCount       int    `csv:"dot-count"`
	class          string `csv:"class"`
}

func calculateFeatures(s scraper.Scraper, featuresChan chan<- *Features, wordMap map[string]bool, globalConfig *scraper.GlobalConfig, wg *sync.WaitGroup) {
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
			f := Features{
				digitCount:     countDigits(fValueString),
				runeCount:      countRunes(fValueString),
				dictWordsCount: countDictWords(fValueString, wordMap),
				slashCount:     countRune(fValueString, []rune("/")[0]),
				colonCount:     countRune(fValueString, []rune(":")[0]),
				dashCount:      countRune(fValueString, []rune("-")[0]),
				dotCount:       countRune(fValueString, []rune(".")[0]),
				class:          fName,
			}
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
		go calculateFeatures(s, featuresChan, wordMap, &config.Global, &calcWg)
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
	return nil
}

func loadWords() (map[string]bool, error) {
	words := map[string]bool{}
	file, err := os.Open("word-lists/english.dic")
	defer file.Close()
	if err != nil {
		return words, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		words[strings.ToLower(scanner.Text())] = true
	}
	return words, nil
}
