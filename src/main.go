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
)

var fileName, urlPrefix, inputFileFormat, fileRegEx, directoryName string;
var maxUrls int;
var showHits, showStatistics, fullDisplay, aggregateData bool;

type Key string
type HitCount int

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
func parseLine (line string) string {

    switch inputFileFormat {
        case "nginx" : {
            url := strings.Split(strings.Split(line, "uri=")[1], " ref=")[0];
            return url[1:len(url)-1];
        };

        case "cloudfront" : {
            return strings.Split(line, "\t")[7];
        }
    }

    return "";
}

func init() {
    flag.StringVar(&inputFileFormat, "fmt", "nginx", "Set the input file format. Can be: nginx, apache, cloudfront. Default: nginx");

    flag.StringVar(&fileName, "f", "", "Name of the file to be parsed, default empty.");

    flag.StringVar(&fileRegEx, "fr", ".*", "Regex of the files to be parsed, default (.*) all. If used it will override -f. Must be used with -dir option.");

    flag.StringVar(&directoryName, "dir", "", "Directory name of where the files should be loaded from, default empty. If used it will override -f. Must be used with -fr option.");

    flag.IntVar(&maxUrls, "l", 0, "Number of lines to be parsed, default, all, 0 = all.");

    flag.BoolVar(&showHits, "h", false, "Show the hits for the urls, default false.");

    flag.StringVar(&urlPrefix, "p", "", "Set the prefix for the urls to be displayed, default empty.");

    flag.BoolVar(&showStatistics, "s", false, "Show statistics for hits of the urls, default false.");

    flag.BoolVar(&fullDisplay, "fd", false, "Just extract the whole urls and do nothing else to process them. Overrides all other switches but limit and prefix.");

    flag.BoolVar(&aggregateData, "a", false, "Aggregate data from all input files. Must be used with -dir option.");
}

func displayOutput(urlHits map[Key]HitCount, urlCount int) {

    largestHit := 0;
    largestHitURL := "";

    if (showHits) {
        if (showStatistics) {

            sortedUrls := make(Elems, 0, len(urlHits))

            for key, value := range urlHits {
                if (int(value) > int(largestHit)) {
                    largestHit = int(value);

                    largestHitURL = string(key);
                }

                sortedUrls = append(sortedUrls, &Elem{key, value});
            }

            sort.Sort(ByReverseCount{sortedUrls});

            fmt.Println("Sorted order:");

            for _, sortedUrl := range sortedUrls {
                fmt.Printf("URL %s%s: hits: %d\n", urlPrefix, sortedUrl.Key, sortedUrl.HitCount);
            }

            fmt.Printf("\nBiggest URL: %s%s hits: %d\n", urlPrefix, largestHitURL, largestHit);
            fmt.Printf("Total URLs: %d\n", urlCount);
        } else {
            for key, value := range urlHits {
                fmt.Printf("URL: %s%s hits: %d\n", urlPrefix, key, value);
            }
        }
    } else {
        for key, _ := range urlHits {
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

        files, _ := ioutil.ReadDir(directoryName);

        for _, f := range files {
            if (!f.IsDir() && includeRegex.Match([]byte(f.Name()))) {
                fileList = append(fileList, directoryName + f.Name());
            }
        }

    } else {
        fileList[0] = fileName;
    }

    urlCount := 0;
    urlHits := make(map [Key]HitCount);

    for _, fileName := range fileList {

        f, err := os.Open(fileName);
        if err != nil {
            fmt.Printf("Error opening file: %v\n",err);
            os.Exit(1);
        }

        defer f.Close();

        r := bufio.NewReader(f);

        if (inputFileFormat == "cloudfront") {
            r.ReadLine();
            r.ReadLine();
        }

        if (!aggregateData) {
            fmt.Printf("\n\nAnalyzing file: %s\n", fileName);
        }

        s, _, e := r.ReadLine();
        for e == nil {
            url := parseLine(string(s));
            if (fullDisplay) {
                fmt.Printf("%s%s\n", urlPrefix, url);
            } else {
                urlHits[Key(url)] += 1;
            }

            s, _, e = r.ReadLine();

            urlCount++;
            if (maxUrls != 0 && urlCount > maxUrls) {
                break;
            }
        }
        f.Close();

        if (fullDisplay) {
            return;
        }

        if (!aggregateData) {
            displayOutput(urlHits, urlCount);
            urlCount = 0;
            urlHits = make(map [Key]HitCount);
        }
    }

    if (aggregateData) {
        displayOutput(urlHits, urlCount);
    }

}
