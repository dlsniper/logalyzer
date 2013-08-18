logalyzer
=========

CloudFront and Nginx log analyzer

This was originally created to help parsing the log files of CloudFront and get some statistics out of them.


Parameters
---

Parameter | Description
----|------
```fmt```  | Input file format. Can be: ```nginx```, ```cloudfront```. Default: ```nginx```
```f```    | Name of the file to be parsed, default empty ``` ```
```fr```   | Regex of the files to be parsed, default all ```(.*)```. If used it will override ```-f```. Must be used with ```-dir``` option
```dir```  | Directory name of where the files should be loaded from, default empty ``` ```. If used it will override ```-f```. Must be used with ```-fr``` option
```url```  | Regex of the urls that will be matched. Default all ```(.*)```
```iqs```  | Ignore the query string of the url. Default ```false```
```rt```   | Type of the request: ```GET```, ```POST```, ```PUT``` and so on. If empty, all request will be processed. Default: empty ``` ```
```cfrt``` | Type of the CloudFront request : ```Hit```, ```RefreshHit```, ```Miss```, ```Pass```(```RefreshHit, Miss```), ```LimitExceded```, ```CapacityExceeded```, ```Exceed```(```LimitExceded, CapacityExceeded```), ```Error```. If empty, all request will be processed. Default: empty ``` ```
```l```    | Number of lines to be parsed. Default, all, ```0```
```h```    | Show the hits for the urls. Default ```false```
```p```    | Set the prefix for the urls to be displayed. Default empty ``` ```
```s```    | Compute statistics for hits of the urls. Default ```false```
```hs```   | Show statistics in human format. Default ```true```
```fd```   | Just extract the whole urls and do nothing else to process them. Overrides all other switches but ```limit``` and ```-p```
```a```    | Aggregate data from all input files. Must be used with ```-dir``` option
```af```   | Aggregate data from the chunks of N files. If 0 is passed then all files will be aggregated. This must be used with ```-a```. Default ```0```
```ab```   | Aggregate by: ```url```, ```hm``` (hits/minute), ```uhm``` (url hits / minute for a specific url). Default ```url```
```tu```   | Display only the first N accessed URLs. If 0 is passed then all URLs will be shown. This must be used with ```-s```. Default ```0```
```su```   | Display a separator every Nth accessed URLs. If 0 is passed then all fallback to default. This must be used with ```-s```. Default ```100```
```v```    | Verbose. Default ```false```


Usage
---

- Get the list of all urls sorted by total number of hits ignoring query string
```bash
logalyzer -a=true -fmt="cloudfront" -dir="/path/" -p="http://localhost" -cfrt="Pass" -iqs=true
```

- Get the list of all urls sorted by total number of hits ignoring query string and display the statistics for them, in human readable format
```bash
logalyzer -a=true -fmt="cloudfront" -dir="/path/" -p="http://localhost" -cfrt="Pass" -s -hs=true -iqs=true
```

- Get list of the highest 10 minutes hits for a specified url pattern (phpinfo.php) from CloudFront in human readable format
```bash
logalyzer -a=true -fmt="cloudfront" -dir="/path/" -p="http://localhost" -cfrt="Pass" -s -hs=true -ab="uhm" -url="(?i)/phpinfo\.php" -iqs=true -tu=10
```

Performance
---

This should be highly performant due to GO mostly but it's also depending on the used hardware.
On my machine, Intel Core i7-3635QM, 12GB and SSD for parsing:

type | value
----|------
files | 5382
total size | 14GB
avg load before test | 0.40

using:
```bash
logalyzer -a=true -fmt="cloudfront" -dir="/path/" -p="http://localhost" -s -hs=true -iqs=true
```

type | value
----|------
max load | 1.40
time logalyzer | 200,25s user 5,99s system 97% cpu 3:31,00 total
Total unique URLs | 69321
Total URLs accesed | 50573946

I have the following:
```bash
logalyzer -a=true -fmt="cloudfront" -dir="/path/" -p="http://localhost" -cfrt="Pass" -s -hs=true -iqs=true
```

type | value
----|------
max load | 1.30
time logalyzer | 187,30s user 6,10s system 98% cpu 3:17,06 total
Total unique URLs | 65377
Total URLs accesed | 12243949


Disclaimer
----

Please note that I've generated the above files for demo purposes only and they don't reflect in any way, shape or form any real data that I may or may not have access to.


Thank you
---

A big thank you goes to [motain GmbH](http://motain.de) for allowing this to be open sourced.


Final word
---

Some of the CloudFront analyzers on the Internet can cost a lot of money. If you do happen to use this, please help me improving it by submitting pull requests or just sending me a thank you on Twitter: ```@dlsniper```
If you make changes on your own environment, please stick to the license and add them back here as well. Thanks.
