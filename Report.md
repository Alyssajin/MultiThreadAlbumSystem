# Project Report

## 1. Data Model
We implemented a **MySQL** database running on AWS RDS, which contains a single table named **Album**.

|Column|Type|Description|
|------|----|-----------|
|id|INT AUTO_INCREMENT PRIMARY KEY|Unique album ID|
|artist|VARCHAR(255) NOT NULL|Artist's name|
|title|VARCHAR(255) NOT NULL|Album title|
|year|INT NOT NULL|Release year of the album|
|image|LONGBLOB NOT NULL|Album cover image in binary format|
|image_size|INT NOT NULL|Size of image file in bytes|

*We used the stock image with a size of approximately 25KB.*

## 2. A Single Server

## 3. Two Load Balanced Servers
We used a thread group size of 10 and delay time 2 seconds. 
|Configuration|Load Balanced Server - numThreadGroups 10|Load Balanced Server - numThreadGroups 20|Load Balanced Server -numThreadGroups 30|
|-|-|-|-|
|Wall Time(s)| |232| |
|Throughput(request/s)| |1724| |
|Successful GET Request Number| |149234| |
|Failed GET Request Number| |50766| |
|GET Request Success Rate| |74.617%| |
|Successful POST Request Number| |163533| |
|Failed POST Request Number| |36467| |
|POST Request Success Rate| |81.767%| |
|Overall Request Success Rate| |78.192%| |
|GET Mean Latency| |78.815655| |
|POST Mean Latency| |111.683405| |
|GET Min Latency| |16| |
|POST Min Latency| |25| |
|GET Max Latency| |15211| |
|POST Max Latency| |15361| |
|GET 50th Percentile| |64.0| |
|POST 50th Percentile| |91.0| |
|GET 99th Percentile| |274.0| |
|POST 99th Percentile| |391.0| |



## 4. Optimized Server Configuration
