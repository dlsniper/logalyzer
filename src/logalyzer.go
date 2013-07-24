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
    "fmt"
    "bufio"
    "os"
    "strings"
    "flag"
    "log"
    "sort"
    "io/ioutil"
    "regexp"
    "runtime"
)

var fileName, urlPrefix, inputFileFormat, fileRegEx, directoryName, requestType, cfRequestType string;
var maxUrls, aggregateEveryNthFiles, showOnlyFirstNthUrls, showSeparatorEveryNthUrls uint;
var showHits, showStatistics, showHumanStatistics, fullDisplay, aggregateData, verbose bool;

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

// Get the URL entry, should be changed to something more flexible :)
// @TODO change this
func parseLine (line *string) (string, bool) {

    var match bool = true;

    switch inputFileFormat {
        case "nginx" : {
            url := strings.Split(strings.Split(*line, "uri=")[1], " ref=")[0];

            rt := strings.Split(strings.Split(*line, "method=")[1], " status=")[0];

            switch requestType {
                case "" : {

                }
                default : {
                    match = rt == requestType;
                }
            }

            return url[1:len(url)-1], match;
        };

        case "cloudfront" : {
            splitUrl := strings.Split(*line, "\t");
            if (len(splitUrl) < 14) {
                return "", false;
            }

            cfRT := splitUrl[13];

            switch cfRequestType {
                case "" : {

                }

                case "Pass" : {
                    match = cfRT == "RefreshHit" || cfRT == "Miss";
                }

                case "Exceed" : {
                    match = cfRT == "LimitExceded" || cfRT == "CapacityExceeded";
                }

                default: {
                    match = cfRT == cfRequestType;
                }
            }

            rt := splitUrl[5];

            switch requestType {
                case "" : {

                }
                default : {
                    match = rt == requestType;
                }
            }

            return splitUrl[7], match;
        }
    }

    return "", false;
}

func init() {
    flag.StringVar(&inputFileFormat, "fmt", "nginx", "Set the input file format. Can be: nginx, apache, cloudfront. Default: nginx");

    flag.StringVar(&fileName, "f", "", "Name of the file to be parsed, default empty.");

    flag.StringVar(&fileRegEx, "fr", ".*", "Regex of the files to be parsed, default (.*) all. If used it will override -f. Must be used with -dir option.");

    flag.StringVar(&directoryName, "dir", "", "Directory name of where the files should be loaded from, default empty. If used it will override -f. Must be used with -fr option.");

    flag.StringVar(&requestType, "rt", "", "Type of the request: GET, POST, PUT and so on. If empty, all request will be processed. Default: empty");

    flag.StringVar(&cfRequestType, "cfrt", "", "Type of the CloudFront request : Hit, RefreshHit, Miss, Pass(RefreshHit, Miss), LimitExceded, CapacityExceeded, Exceed(LimitExceded, CapacityExceeded), Error. If empty, all request will be processed. Default: empty");

    flag.UintVar(&maxUrls, "l", 0, "Number of lines to be parsed, default, all, 0 = all.");

    flag.BoolVar(&showHits, "h", false, "Show the hits for the urls, default false.");

    flag.StringVar(&urlPrefix, "p", "", "Set the prefix for the urls to be displayed, default empty.");

    flag.BoolVar(&showStatistics, "s", false, "Compute statistics for hits of the urls, default false.");

    flag.BoolVar(&showHumanStatistics, "hs", true, "Show statistics in human format, default true.");

    flag.BoolVar(&fullDisplay, "fd", false, "Just extract the whole urls and do nothing else to process them. Overrides all other switches but limit and prefix.");

    flag.BoolVar(&aggregateData, "a", false, "Aggregate data from all input files. Must be used with -dir option.");

    flag.UintVar(&aggregateEveryNthFiles, "af", 0, "When this is used, it can aggregate data from the chunks of N files. If 0 is passed then all files will be aggregated. This must be used with -a.");

    flag.UintVar(&showOnlyFirstNthUrls, "tu", 0, "When this is used, it will display only the first N accessed URLs. If 0 is passed then all URLs will be shown. This must be used with -s.");

    flag.UintVar(&showSeparatorEveryNthUrls, "su", 100, "When this is used, it will display will display a separator every Nth accessed URLs. If 0 is passed then all URLs will be shown it will fallback to default, 100. This must be used with -s.");

    flag.BoolVar(&verbose, "v", false, "Verbose. Default no (false)");
}

func displayOutput(urlHits *map[Key]HitCount, urlCount uint) {

    var largestHit uint = 0;
    largestHitURL := "";

    if (showHits) {
        if (showStatistics) {

            uniqueUrlsCount := len(*urlHits);

            sortedUrls := make(Elems, 0, uniqueUrlsCount)

            for key, value := range *urlHits {
                if (uint(value) > uint(largestHit)) {
                    largestHit = uint(value);

                    largestHitURL = string(key);
                }

                sortedUrls = append(sortedUrls, &Elem{key, value});
            }

            *urlHits = make(map [Key]HitCount);

            runtime.GC();

            sort.Sort(ByReverseCount{sortedUrls});

            runtime.GC();

            i:=0;
            for _, sortedUrl := range sortedUrls {
                i++;

                if (showHumanStatistics) {
                    fmt.Printf("%d URL %s%s: hits: %d\n", i, urlPrefix, sortedUrl.Key, sortedUrl.HitCount);
                } else {
                    fmt.Printf("%s%s\n", urlPrefix, sortedUrl.Key);
                }

                if (uint(i) == showOnlyFirstNthUrls) {
                    break;
                }

                if (uint(i) % showSeparatorEveryNthUrls == 0) {
                    if (showSeparatorEveryNthUrls == 0) {
                        fmt.Printf("============ %d/%d ===========================================================\n", i, uniqueUrlsCount);
                    } else {
                        fmt.Printf("============ %d/%d (%d total)=================================================\n", i, showOnlyFirstNthUrls, uniqueUrlsCount);
                    }
                }
            }

            if (showHumanStatistics) {
                fmt.Printf("\nBiggest URL: %s%s hits: %d\n", urlPrefix, largestHitURL, largestHit);
                fmt.Printf("Total unique URLs: %d\n", uniqueUrlsCount);
                fmt.Printf("Total URLs: %d\n", urlCount);
            }
        } else {
            for key, value := range *urlHits {
                fmt.Printf("URL: %s%s hits: %d\n", urlPrefix, key, value);
            }
        }
    } else {
        for key, _ := range *urlHits {
            fmt.Printf("%s%s\n", urlPrefix, key);
        }
    }
}

func main() {

    flag.Parse();

    if (fileName == "" && directoryName == "" && fileRegEx == "")  {
        log.Fatalln("Filename or directory not specified");
    }

    if (showStatistics) {
        showHits = true;
    }

    var fileList []string;

    if (fileName == "") {
        includeRegex, err := regexp.Compile(fileRegEx);
        if err != nil {
            log.Fatal(err);
        }

        files, err := ioutil.ReadDir(directoryName);
        if (err != nil) {
            log.Fatal(err);
        }

        for _, f := range files {
            if (!f.IsDir() && includeRegex.Match([]byte(f.Name()))) {
                fileList = append(fileList, directoryName + f.Name());
            }

        }

        files = []os.FileInfo{};

        runtime.GC();
    } else {
        fileList[0] = fileName;
    }

    var urlCount uint = 0;
    urlHits := make(map [Key]HitCount);

    var url, line string;
    var valid bool;
    var fileNumber uint = 0;
    fileCount := len(fileList);

    for _, fileName := range fileList {
        fileNumber++;

        if (verbose) {
            fmt.Printf("%d/%d file: %s\n", fileNumber, fileCount, fileName);
        }

        f, err := os.Open(fileName);
        if err != nil {
            log.Fatalf("Error opening file: %v\n",err);
        }

        r := bufio.NewReader(f);

        // @TODO change this
        if (inputFileFormat == "cloudfront") {
            r.ReadLine();
            r.ReadLine();
        }

        if (!aggregateData && verbose) {
            fmt.Printf("\n\nAnalyzing file: %s\n", fileName);
        }

        s, _, e := r.ReadLine();
        for e == nil {
            line = string(s);
            url, valid = parseLine(&line);

            s, _, e = r.ReadLine();

            if (!valid) {
                continue;
            }

            if (fullDisplay) {
                fmt.Printf("%s%s\n", urlPrefix, url);
            } else {
                urlHits[Key(url)] += 1;
            }

            urlCount++;
            if (maxUrls != 0 && urlCount > maxUrls) {
                break;
            }
        }
        f.Close();

        if (fullDisplay) {
            continue;
        }

        if (!aggregateData) {
            displayOutput(&urlHits, urlCount);
            urlCount = 0;
            urlHits = make(map [Key]HitCount);
        } else if (aggregateEveryNthFiles != 0) {
            if (fileNumber % aggregateEveryNthFiles == 0) {
                displayOutput(&urlHits, urlCount);
                urlCount = 0;
                urlHits = make(map [Key]HitCount);
            }
        }

        runtime.GC();
    }

    fileList = []string{};

    runtime.GC();

    if (aggregateData) {
        displayOutput(&urlHits, urlCount);
    }

}
