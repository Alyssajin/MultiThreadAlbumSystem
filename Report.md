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
|-----------------------------|----------|----------|----------|
|Wall Time(s)                 | 192      |       232|   331     |
|Throughput(request/s)        |   1041   |   1724   | 1812|
|Successful GET Request Number|67193     |149234    |191740 |
|Failed GET Request Number    |32807     |50766     |108260 |
|GET Request Success Rate     |  67.193% |74.617%   | 63.913%|
|Successful POST Request Number|73314    |163533    | 200377|
|Failed POST Request Number   |  26686   |  36467   | 99623|
|POST Request Success Rate    | 73.314%  | 81.767%  | 66.792%|
|Overall Request Success Rate|70.254%    |78.192%| 65.353%|
|GET Mean Latency            |65.30091   |78.815655| 84.058603|
|POST Mean Latency         |105.68013 |111.683405|191.24366 |
|GET Min Latency|16 |16| 16|
|POST Min Latency|23 |25|22 |
|GET Max Latency|1235 |15211|16296 |
|POST Max Latency|1668 |15361| 16584|
|GET 50th Percentile| 57.0|64.0| 57.0|
|POST 50th Percentile| 83.0|91.0| 157.0|
|GET 99th Percentile| 182.0|274.0| 1062.0|
|POST 99th Percentile|433.0 |391.0| 1140.0|



## 4. Optimized Server Configuration
