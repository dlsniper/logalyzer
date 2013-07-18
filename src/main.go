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
)

var fileName, urlPrefix, inputFileFormat string;
var maxUrls int;
var showHits, showStatistics, fullDisplay bool;

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

    flag.StringVar(&fileName, "f", "", "Set the name of the file to be parsed, default empty");

    flag.IntVar(&maxUrls, "l", 0, "Set the number of lines to be parsed, default, all, 0 = all");

    flag.BoolVar(&showHits, "h", false, "Show the hits for the urls, default false");

    flag.StringVar(&urlPrefix, "p", "", "Set the prefix for the urls to be displayed, default empty");

    flag.BoolVar(&showStatistics, "s", false, "Show statistics for hits of the urls, default false");

    flag.BoolVar(&fullDisplay, "fd", false, "Just extract the whole urls and do nothing else to process them. Overrides all other switches but limit and prefix");
}

func main() {

    flag.Parse();

    if fileName == "" {
        log.Fatalln("Filename not specified");
    }

    urlHits := make(map [Key]HitCount);

    f, err := os.Open(fileName);
    if err != nil {
        fmt.Printf("Error opening file: %v\n",err);
        os.Exit(1);
    }
    defer f.Close();
    r := bufio.NewReader(f);

    largestHit := 0;
    largestHitURL := "";

    if (inputFileFormat == "cloudfront") {
        r.ReadLine();
        r.ReadLine();
    }

    s, _, e := r.ReadLine();
    urlCount := 0;
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
