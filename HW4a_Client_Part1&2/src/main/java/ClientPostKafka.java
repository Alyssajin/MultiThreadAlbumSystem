import org.apache.kafka.clients.producer.KafkaProducer;
import org.apache.kafka.clients.producer.ProducerRecord;
import org.apache.kafka.clients.producer.RecordMetadata;
import org.apache.spark.sql.Row;
import org.apache.spark.sql.RowFactory;

import java.io.File;
import java.io.IOException;
import java.nio.file.Files;
import java.util.Base64;
import java.util.List;
import java.util.Properties;
import java.util.concurrent.Future;
import java.util.concurrent.atomic.AtomicInteger;

public class ClientPostKafka implements Runnable {
  private static AtomicInteger successCount = new AtomicInteger(0);
  private static AtomicInteger failCount = new AtomicInteger(0);

  private KafkaProducer<String, byte[]> producer;
  private String topic;
  private List<Row> data;
  private File file;

  // Constructor: bootstrapServers should be something like "localhost:9092" or your broker's endpoint.
  public ClientPostKafka(String bootstrapServers, String topic, List<Row> data, File file) {
    this.topic = topic;
    this.data = data;
    this.file = file;
    Properties props = new Properties();
    props.put("bootstrap.servers", bootstrapServers);
    props.put("key.serializer", "org.apache.kafka.common.serialization.StringSerializer");
    props.put("value.serializer", "org.apache.kafka.common.serialization.ByteArraySerializer");
    this.producer = new KafkaProducer<>(props);
  }

  @Override
  public void run() {
    long startTime = System.currentTimeMillis();
    System.out.println("Kafka POST START: " + startTime + " " + Thread.currentThread().getName());
    try {
      // Read file contents as bytes
      byte[] fileBytes = Files.readAllBytes(file.toPath());
      // Encode the file bytes as Base64 so we can include them in a JSON string
      String base64File = Base64.getEncoder().encodeToString(fileBytes);
      // Create a JSON string for the profile field
      String jsonProfile = "{\"artist\":\"AgustD\",\"title\":\"D-Day\",\"year\":\"2023\"}";
      // Combine the profile and file into one JSON message.
      // You might structure this differently depending on your needs.
      String combinedMessage = "{\"profile\":" + jsonProfile + ", \"file\":\"" + base64File + "\"}";

      // Create a ProducerRecord. Using "my-key" as key; you can adjust as needed.
      ProducerRecord<String, byte[]> record = new ProducerRecord<>(topic, "my-key", combinedMessage.getBytes("UTF-8"));

      // Send record asynchronously and wait for acknowledgement.
      Future<RecordMetadata> future = producer.send(record);
      RecordMetadata metadata = future.get();

      // If send is successful, update our counters and record latency.
      successCount.incrementAndGet();
      int statusCode = 200;  // Simulate success (Kafka doesn't use HTTP status codes)
      long endTime = System.currentTimeMillis();
      long latency = endTime - startTime;
      data.add(RowFactory.create(startTime, "POST", latency, statusCode));
      System.out.println("Kafka POST END: " + endTime + " " + Thread.currentThread().getName());
    } catch (Exception e) {
      failCount.incrementAndGet();
      System.err.println("Kafka POST error: " + e.getMessage());
      long endTime = System.currentTimeMillis();
      long latency = endTime - startTime;
      data.add(RowFactory.create(startTime, "POST", latency, 500));
    }
  }

  public static int getSuccessCount() {
    return successCount.get();
  }

  public static int getFailCount() {
    return failCount.get();
  }

  public void close() {
    producer.close();
  }
}