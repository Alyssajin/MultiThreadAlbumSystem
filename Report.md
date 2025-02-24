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

## 3. Two Load Balanced Server

## 4. Optimized Server Configuration
