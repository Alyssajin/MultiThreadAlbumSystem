import java.io.File;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.concurrent.CountDownLatch;

import org.apache.hc.client5.http.impl.DefaultHttpRequestRetryStrategy;
import org.apache.hc.client5.http.impl.async.CloseableHttpAsyncClient;
import org.apache.hc.client5.http.impl.async.HttpAsyncClients;
import org.apache.hc.client5.http.impl.classic.CloseableHttpClient;
import org.apache.hc.client5.http.impl.classic.HttpClients;
import org.apache.hc.client5.http.impl.io.PoolingHttpClientConnectionManager;
import org.apache.hc.client5.http.impl.nio.PoolingAsyncClientConnectionManager;
import org.apache.hc.core5.util.TimeValue;
import org.apache.spark.sql.Dataset;
import org.apache.spark.sql.Row;
import org.apache.spark.sql.SparkSession;
import org.apache.spark.sql.types.StructType;


public class Main {

  public static void main(String[] args) throws InterruptedException {
    if (args.length != 4) {
      System.out.println(
          "Usage: java HW4aServer <threadGroupSize> <numThreadGroups> <delay> <IPAddr>");
      System.exit(1);
    }
    int groupSize = Integer.parseInt(args[0]);
    int numGroups = Integer.parseInt(args[1]);
    int delay = Integer.parseInt(args[2]);
    String IPAddr = args[3];

    PoolingHttpClientConnectionManager connectionManager = new PoolingHttpClientConnectionManager();
    connectionManager.setMaxTotal(1000);
    connectionManager.setDefaultMaxPerRoute(1000);
    // Create an instance of HttpClient.

    CloseableHttpClient client = HttpClients.custom().setConnectionManager(connectionManager)
        .setRetryStrategy(new DefaultHttpRequestRetryStrategy(5, TimeValue.ofSeconds(2))).build();

    try {
      SparkSession spark =
          SparkSession.builder().appName("data").config("spark.master", "local").getOrCreate();
      spark.sparkContext().setLogLevel("ERROR");
      List<Row> list = Collections.synchronizedList(new ArrayList<>());
      File file = new File("/Users/boyuansun/Desktop/25Spring/CS 6650/Example.jpg");
      ClientGet clientGet = new ClientGet(IPAddr, client, list);
      ClientPost clientPost = new ClientPost(IPAddr, client, list, file);
      CountDownLatch completed = new CountDownLatch(10);
      for (int i = 0; i < 10; i++) {
        // lambda runnable creation - interface only has a single method so lambda works fine
        Runnable thread = () -> {
          for (int j = 0; j < 100; j++) {
            clientGet.run();
            clientPost.run();
          }
          completed.countDown();
        };
        new Thread(thread).start();

      }

      completed.await();

      int totalThreads = groupSize * numGroups;

      CountDownLatch completed2 = new CountDownLatch(totalThreads);
      long start = System.currentTimeMillis();
      for (int j = 0; j < numGroups; j++) {
        for (int i = 0; i < groupSize; i++) {
          Runnable thread = () -> {
            for (int k = 0; k < 1000; k++) {
              clientGet.run();
              clientPost.run();
            }
            completed2.countDown();
            //System.out.println("Thread completed");
          };
          new Thread(thread).start();
          //System.out.println("Thread started");
        }
        if (j != numGroups - 1) {
          Thread.sleep(delay * 1000);
        }
      }
      completed2.await();

      long end = System.currentTimeMillis();
      spark.stop();
      System.out.println("Time taken: " + (end - start) / 1000 + "s");
      System.out.println(
          "Throughput: " + ((totalThreads * 2000)) / ((end - start) / 1000) +
              " requests/s");
    } catch (Exception e) {
      e.printStackTrace();
    }
  }
}
