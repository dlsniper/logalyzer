/**
 * This file is part of the logalyzer package.
 *
 * (c) Florin Patan <florinpatan@gmail.com>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

var fileName, urlPrefix, inputFileFormat, fileRegEx, urlRegEx, directoryName, requestType, cfRequestType, aggregateBy string
var maxUrls, aggregateEveryNthFiles, showOnlyFirstNthUrls, showSeparatorEveryNthUrls uint
var showHits, showStatistics, showHumanStatistics, fullDisplay, aggregateData, verbose, ignoreQueryString bool
var compiledFileRegEx, compiledUrlRegEx *regexp.Regexp

type Key string
type HitCount uint

type Elem struct {
	Key
	HitCount
}

type Elems []*Elem

func (s Elems) Len() int {
	return len(s)
}

func (s Elems) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type ByReverseCount struct {
	Elems
}

func (s ByReverseCount) Less(i, j int) bool {
	return s.Elems[j].HitCount < s.Elems[i].HitCount
}

func init() {
	flag.StringVar(&inputFileFormat, "fmt", "nginx", "Set the input file format. Can be: nginx, apache, cloudfront. Default: nginx")

	flag.StringVar(&fileName, "f", "", "Name of the file to be parsed, default empty.")

	flag.StringVar(&fileRegEx, "fr", ".*", "Regex of the files to be parsed, default all (.*). If used it will override -f. Must be used with -dir option.")

	flag.StringVar(&directoryName, "dir", "", "Directory name of where the files should be loaded from, default empty. If used it will override -f. Must be used with -fr option.")

	flag.StringVar(&urlRegEx, "url", ".*", "Regex of the urls that will be matched, default all (.*)")

	flag.BoolVar(&ignoreQueryString, "iqs", false, "Ignore the query string of the url, default false.")

	flag.StringVar(&requestType, "rt", "", "Type of the request: GET, POST, PUT and so on. If empty, all request will be processed. Default: empty")

	flag.StringVar(&cfRequestType, "cfrt", "", "Type of the CloudFront request : Hit, RefreshHit, Miss, Pass(RefreshHit, Miss), LimitExceded, CapacityExceeded, Exceed(LimitExceded, CapacityExceeded), Error. If empty, all request will be processed. Default: empty")

	flag.UintVar(&maxUrls, "l", 0, "Number of lines to be parsed, default, all, 0 = all.")

	flag.BoolVar(&showHits, "h", false, "Show the hits for the urls, default false.")

	flag.StringVar(&urlPrefix, "p", "", "Set the prefix for the urls to be displayed, default empty.")

	flag.BoolVar(&showStatistics, "s", false, "Compute statistics for hits of the urls, default false.")

	flag.BoolVar(&showHumanStatistics, "hs", true, "Show statistics in human format, default true.")

	flag.BoolVar(&fullDisplay, "fd", false, "Just extract the whole urls and do nothing else to process them. Overrides all other switches but limit and prefix.")

	flag.BoolVar(&aggregateData, "a", false, "Aggregate data from all input files. Must be used with -dir option.")

	flag.UintVar(&aggregateEveryNthFiles, "af", 0, "When this is used, it can aggregate data from the chunks of N files. If 0 is passed then all files will be aggregated. This must be used with -a.")

	flag.StringVar(&aggregateBy, "ab", "url", "Aggregate by: url, hm (hits/minute), uhm (url hits / minute for a specific url). Default url")

	flag.UintVar(&showOnlyFirstNthUrls, "tu", 0, "When this is used, it will display only the first N accessed URLs. If 0 is passed then all URLs will be shown. This must be used with -s.")

	flag.UintVar(&showSeparatorEveryNthUrls, "su", 100, "When this is used, it will display will display a separator every Nth accessed URLs. If 0 is passed then all URLs will be shown it will fallback to default, 100. This must be used with -s.")

	flag.BoolVar(&verbose, "v", false, "Verbose. Default no (false)")
}

func parseNginxLine(line *string) (string, int64, bool) {
	var match bool = true

	stringUrl := strings.Split(strings.Split(*line, "uri=")[1], " ref=")[0]

	if !compiledUrlRegEx.Match([]byte(stringUrl)) {
		return "", 0, false
	}

	rt := strings.Split(strings.Split(*line, "method=")[1], " status=")[0]

	switch requestType {
	case "":
		{

		}
	default:
		{
			match = rt == requestType
		}
	}

	return stringUrl[1 : len(stringUrl)-1], 0, match
}

func parseCloudfrontLine(line *string) (string, int64, bool) {
	var match bool = true

	splitUrl := strings.Split(*line, "\t")
	if len(splitUrl) < 14 {
		return "", 0, false
	}

	cfRT := splitUrl[13]

	if urlRegEx != "" && urlRegEx != ".*" && !compiledUrlRegEx.Match([]byte(splitUrl[7])) {
		return "", 0, false
	}

	stringUrl := splitUrl[7]

	if stringUrl[len(stringUrl) - 1:] != "/" {
		if stringUrl[len(stringUrl) - 4:len(stringUrl) - 3] != "." &&
			stringUrl[len(stringUrl) - 5:len(stringUrl) - 4] != "." {
			stringUrl += "/"
		}
	}

	if splitUrl[11] != "-" {
		stringUrl += "?" + splitUrl[11]
	}

	if ignoreQueryString {
		parsedUrl, err := url.Parse(urlPrefix + stringUrl)
		if err != nil {
			return "", 0, false
		}

		stringUrl = strings.Replace(stringUrl, "?"+parsedUrl.RawQuery+"#"+parsedUrl.Fragment, "", -1)
		stringUrl = strings.Replace(stringUrl, "?"+parsedUrl.RawQuery, "", -1)
	}

	switch cfRequestType {
	case "":
		{

		}

	case "Pass":
		{
			match = cfRT == "RefreshHit" || cfRT == "Miss"
		}

	case "Exceed":
		{
			match = cfRT == "LimitExceded" || cfRT == "CapacityExceeded"
		}

	default:
		{
			match = cfRT == cfRequestType
		}
	}

	rt := splitUrl[5]

	switch requestType {
	case "":
		{

		}
	default:
		{
			match = rt == requestType
		}
	}

	var requestTimestamp int64 = 0

	switch aggregateBy {
	case "url":
		{
			requestTimestamp = 0
		}
	case "uhm":
		{
			requestTime, err := time.Parse("2006-01-02 15:04", fmt.Sprintf("%s %s", splitUrl[0], splitUrl[1][0:5]))
			if err != nil {
				requestTimestamp = 0
			} else {
				requestTimestamp = requestTime.Unix()
			}
		}
	case "hm":
		{
			requestTime, err := time.Parse("2006-01-02 15:04", fmt.Sprintf("%s %s", splitUrl[0], splitUrl[1][0:5]))
			if err != nil {
				requestTimestamp = 0
			} else {
				requestTimestamp = requestTime.Unix()
			}
		}
	}

	return stringUrl, requestTimestamp, match
}

func parseLine(line *string) (string, int64, bool) {

	switch inputFileFormat {
	case "nginx":
		{
			return parseNginxLine(line)
		}

	case "cloudfront":
		{
			return parseCloudfrontLine(line)
		}
	}

	return "", 0, false
}

func showHumanStatsLine(i int64, line *Elem) {
	switch aggregateBy {
	case "url":
		{
			fmt.Printf("%d URL %s%s: hits: %d\n", i, urlPrefix, line.Key, line.HitCount)
		}
	case "uhm":
		{
			fmt.Printf("%d URL %s%s hits: %d\n", i, urlPrefix, line.Key, line.HitCount)
		}
	case "hm":
		{
			fmt.Printf("%d Time: %s  hits: %d\n", i, line.Key, line.HitCount)
		}
	}
}

func showStatsLine(line *Elem) {
	switch aggregateBy {
	case "url":
		{
			fmt.Printf("%s%s\n", urlPrefix, line.Key)
		}
	case "uhm":
		{
			fmt.Printf("%s%s  %d\n", urlPrefix, line.Key, line.HitCount)
		}
	case "hm":
		{
			fmt.Printf("%s  %d\n", line.Key, line.HitCount)
		}
	}
}

func sortUrls(urlHits *map[Key]HitCount) (Elems, uint64, string) {
	uniqueUrlsCount := len(*urlHits)
	sortedUrls := make(Elems, 0, uniqueUrlsCount)

	var largestHit uint64 = 0
	var largestHitURL string = ""

	if showHumanStatistics && aggregateBy == "url" {
		for key, value := range *urlHits {

			if uint64(value) > uint64(largestHit) {
				largestHit = uint64(value)

				largestHitURL = string(key)
			}

			sortedUrls = append(sortedUrls, &Elem{key, value})
		}
	} else {
		for key, value := range *urlHits {
			sortedUrls = append(sortedUrls, &Elem{key, value})
		}
	}

	*urlHits = make(map[Key]HitCount)

	runtime.GC()

	sort.Sort(ByReverseCount{sortedUrls})

	runtime.GC()

	return sortedUrls, largestHit, largestHitURL
}

func displayOutput(urlHits *map[Key]HitCount, urlCount uint) {

	if !showHits {
		for key, _ := range *urlHits {
			fmt.Printf("%s%s\n", urlPrefix, key)
		}

		return
	}

	if !showStatistics {
		for key, value := range *urlHits {
			fmt.Printf("URL: %s%s hits: %d\n", urlPrefix, key, value)
		}

		return
	}

	uniqueUrlsCount := len(*urlHits)

	sortedUrls, largestHit, largestHitURL := sortUrls(urlHits)

	var i int64 = 0
	for _, sortedUrl := range sortedUrls {
		i++

		if showHumanStatistics {
			showHumanStatsLine(i, sortedUrl)
		} else {
			showStatsLine(sortedUrl)
		}

		if uint(i) == showOnlyFirstNthUrls {
			break
		}

		if uint(i)%showSeparatorEveryNthUrls == 0 {
			if showSeparatorEveryNthUrls == 0 {
				fmt.Printf("============ %d/%d ===========================================================\n", i, uniqueUrlsCount)
			} else {
				fmt.Printf("============ %d/%d (%d total)=================================================\n", i, showOnlyFirstNthUrls, uniqueUrlsCount)
			}
		}
	}

	if showHumanStatistics && aggregateBy == "url" {
		fmt.Printf("\nBiggest URL: %s%s hits: %d\n", urlPrefix, largestHitURL, largestHit)
		fmt.Printf("Total unique URLs: %d\n", uniqueUrlsCount)
		fmt.Printf("Total URLs accesed: %d\n", urlCount)
	}
}

func checkArguments() {
	var err error

	if fileName == "" && directoryName == "" && fileRegEx == "" {
		log.Fatalln("Filename or directory not specified")
	}

	if showStatistics {
		showHits = true
	}

	if urlRegEx != "" && urlRegEx != ".*" {
		compiledUrlRegEx, err = regexp.Compile(urlRegEx)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func getFileList() []string {
	var fileList []string
	var err error

	if fileName == "" {
		compiledFileRegEx, err = regexp.Compile(fileRegEx)
		if err != nil {
			log.Fatal(err)
		}

		files, err := ioutil.ReadDir(directoryName)
		if err != nil {
			log.Fatal(err)
		}

		for _, f := range files {
			if !f.IsDir() && compiledFileRegEx.Match([]byte(f.Name())) {
				fileList = append(fileList, directoryName+f.Name())
			}

		}

		files = []os.FileInfo{}

		runtime.GC()
	} else {
		fileList[0] = fileName
	}

	return fileList
}

func main() {

	flag.Parse()

	checkArguments()

	fileList := getFileList()

	var urlCount uint = 0
	var url, line string
	var requestTimestamp int64
	var valid bool
	var fileNumber uint = 0
	fileCount := len(fileList)
	urlHits := make(map[Key]HitCount)

	for _, fileName := range fileList {
		fileNumber++

		if verbose {
			fmt.Printf("%d/%d file: %s\n", fileNumber, fileCount, fileName)
		}

		f, err := os.Open(fileName)
		if err != nil {
			log.Fatalf("Error opening file: %v\n", err)
		}
		defer f.Close()

		r := bufio.NewReader(f)

		if inputFileFormat == "cloudfront" {
			r.ReadLine()
			r.ReadLine()
		}

		if !aggregateData && verbose {
			fmt.Printf("\n\nAnalyzing file: %s\n", fileName)
		}

		s, _, e := r.ReadLine()
		for e == nil {
			line = string(s)
			url, requestTimestamp, valid = parseLine(&line)

			s, _, e = r.ReadLine()

			if !valid {
				continue
			}

			if fullDisplay {
				fmt.Printf("%s%s\n", urlPrefix, url)
			} else {
				switch aggregateBy {
				case "url":
					{
						urlHits[Key(url)] += 1
					}
				case "hm":
					{
						if showHumanStatistics {
							urlHits[Key(time.Unix(requestTimestamp, 0).Format(time.RFC822))] += 1
						} else {
							urlHits[Key(fmt.Sprintf("%d", requestTimestamp))] += 1
						}
					}
				case "uhm":
					{
						if showHumanStatistics {
							urlHits[Key(url+" on "+time.Unix(requestTimestamp, 0).Format(time.RFC822))] += 1
						} else {
							urlHits[Key(url+" "+fmt.Sprintf("%d", requestTimestamp))] += 1
						}
					}
				}
			}

			urlCount++
			if maxUrls != 0 && urlCount > maxUrls {
				break
			}
		}
		f.Close()

		if fullDisplay {
			runtime.GC()

			continue
		}

		if !aggregateData {
			displayOutput(&urlHits, urlCount)
			urlCount = 0
			urlHits = make(map[Key]HitCount)
		} else if aggregateEveryNthFiles != 0 {
			if fileNumber%aggregateEveryNthFiles == 0 {
				displayOutput(&urlHits, urlCount)
				urlCount = 0
				urlHits = make(map[Key]HitCount)
			}
		}

		runtime.GC()
	}

	fileList = []string{}

	runtime.GC()

	if aggregateData {
		displayOutput(&urlHits, urlCount)
	}

}
