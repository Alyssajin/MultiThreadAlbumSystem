import org.apache.kafka.clients.consumer.ConsumerRecords;
import org.apache.kafka.clients.consumer.KafkaConsumer;
import org.apache.kafka.common.errors.WakeupException;

import java.time.Duration;
import java.util.Arrays;
import java.util.Properties;
import java.util.concurrent.atomic.AtomicBoolean;

public class ClientGetKafka implements Runnable {
  private final AtomicBoolean closed = new AtomicBoolean(false);
  private KafkaConsumer consumer;
  private String topic = "album-get";
  private int albumId;

  public ClientGetKafka(int albumId) {
    this.albumId = albumId;
  }
  public void run() {
    try {
      Properties props = new Properties();
      props.put("bootstrap.servers", "localhost:9092");
      props.put("group.id", "test");
      props.put("enable.auto.commit", "true");
      props.put("auto.commit.interval.ms", "1000");
      props.put("key.deserializer", "org.apache.kafka.common.serialization.StringDeserializer");
      props.put("value.deserializer", "org.apache.kafka.common.serialization.StringDeserializer");
      KafkaConsumer<String, String> consumer = new KafkaConsumer<>(props);

      consumer.subscribe(Arrays.asList(topic));
      ConsumerRecords records = consumer.poll(Duration.ofMillis(10000));
      // Handle new records
      System.out.println("Kafka Received records: " + records.count());
    } catch (WakeupException e) {
      // Ignore exception if closing
      if (!closed.get()) throw e;
      System.err.println("WakeupException");
    } finally {
      if (consumer != null) {
        consumer.close();
      }
    }
  }

  // Shutdown hook which can be called from a separate thread
  public void shutdown() {
    closed.set(true);
    consumer.wakeup();
  }
}



//import org.apache.kafka.clients.consumer.ConsumerRecord;
//import org.apache.kafka.clients.consumer.ConsumerRecords;
//import org.apache.kafka.clients.consumer.KafkaConsumer;
//import org.apache.spark.sql.Row;
//import org.apache.spark.sql.RowFactory;
//
//import java.time.Duration;
//import java.util.Collections;
//import java.util.List;
//import java.util.Properties;
//import java.util.concurrent.atomic.AtomicInteger;
//
//public class ClientGetKafka implements Runnable {
//  private static AtomicInteger successCount = new AtomicInteger(0);
//  private static AtomicInteger failCount = new AtomicInteger(0);
//
//  private KafkaConsumer<String, String> consumer;
//  private String topic;
//  private List<Row> data;
//  private String albumId;
//
//  // Constructor to initialize the Kafka consumer with properties.
//  public ClientGetKafka(String bootstrapServers, String topic, List<Row> data, String albumId) {
//    this.topic = topic;
//    this.data = data;
//    this.albumId = albumId;
//    Properties props = new Properties();
//    props.put("bootstrap.servers", bootstrapServers);
//    props.put("group.id", "client-get-group");
//    props.put("key.deserializer", "org.apache.kafka.common.serialization.StringDeserializer");
//    props.put("value.deserializer", "org.apache.kafka.common.serialization.StringDeserializer");
//    // "earliest" ensures you read from the beginning if no offset is committed.
//    props.put("auto.offset.reset", "earliest");
//    this.consumer = new KafkaConsumer<>(props);
//    consumer.subscribe(Collections.singletonList(topic));
//  }
//
//  @Override
//  public void run() {
//    long start = System.currentTimeMillis();
//    System.out.println("Kafka GET START: " + start + " " + Thread.currentThread().getName());
//    try {
//      // Poll the topic for messages for 1 second.
//      ConsumerRecords<String, String> records = consumer.poll(Duration.ofMillis(5000));
//      if (records != null) {
//        failCount.incrementAndGet();
//        System.err.println("No records consumed");
//      } else
//      if (records.count() > 0) {
//        successCount.incrementAndGet();
//        for (ConsumerRecord<String, String> record : records) {
//          System.out.printf("Consumed record: key=%s, value=%s, offset=%d%n",
//                  record.key(), record.value(), record.offset());
//        }
//      } else {
//        failCount.incrementAndGet();
//        System.err.println("No records consumed");
//      }
//      long end = System.currentTimeMillis();
//      System.out.println("Kafka GET END: " + end + " " + Thread.currentThread().getName());
//      long latency = end - start;
//      // We'll assume a 200 code if messages were consumed, else 500.
//      int code = records.count() > 0 ? 200 : 500;
//      data.add(RowFactory.create(start, "GET", latency, code));
//    } catch (Exception e) {
//      failCount.incrementAndGet();
//      System.err.println("Error in Kafka consumer: " + e.getMessage());
//      long end = System.currentTimeMillis();
//      long latency = end - start;
//      data.add(RowFactory.create(start, "GET", latency, 500));
//    }
//  }
//
//  public void close() {
//    consumer.close();
//  }
//
//  public static int getSuccessCount() {
//    return successCount.get();
//  }
//
//  public static int getFailCount() {
//    return failCount.get();
//  }
//}