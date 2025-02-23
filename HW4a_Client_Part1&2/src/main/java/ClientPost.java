import java.io.File;
import java.io.IOException;
import java.util.List;
import java.util.concurrent.atomic.AtomicInteger;

import org.apache.hc.client5.http.classic.methods.HttpPost;
import org.apache.hc.client5.http.entity.mime.HttpMultipartMode;
import org.apache.hc.client5.http.entity.mime.MultipartEntityBuilder;
import org.apache.hc.client5.http.impl.classic.CloseableHttpClient;
import org.apache.hc.client5.http.impl.classic.CloseableHttpResponse;
import org.apache.hc.core5.http.ContentType;
import org.apache.hc.core5.http.HttpEntity;
import org.apache.hc.core5.http.HttpStatus;
import org.apache.hc.core5.http.io.entity.EntityUtils;
import org.apache.spark.sql.Row;
import org.apache.spark.sql.RowFactory;

public class ClientPost implements Runnable {
  private static AtomicInteger successCount = new AtomicInteger(0);
  private static AtomicInteger failCount = new AtomicInteger(0);
  private String postUrl;
  private CloseableHttpClient client;
  private List<Row> data;
  private File file;


  public ClientPost(String IPAddr, CloseableHttpClient client, List<Row> data, File file) {
    this.postUrl = "http://" + IPAddr + "/album";
    this.client = client;
    this.data = data;
    this.file = file;
  }

  // stolen from https://hc.apache.org/httpclient-legacy/tutorial.html
  public void run() {
    long starttime = System.currentTimeMillis();
    System.out.println("POST START: " + starttime + Thread.currentThread().getName());
    MultipartEntityBuilder builder = MultipartEntityBuilder.create();
    builder.setMode(HttpMultipartMode.STRICT);

    // 1. Add the file part
    builder.addBinaryBody(
            "image",
            file,
            ContentType.IMAGE_JPEG,
            "Example.jpg"
    );

    // 2. Add the profile field as a single JSON string
    // Example JSON: {"artist":"AgustD","title":"D-Day","year":"2023"}
    // Modify these values or pass them in as parameters if needed
    String jsonProfile = "{\"artist\":\"AgustD\",\"title\":\"D-Day\",\"year\":\"2023\"}";

    // Add the 'profile' field with JSON content
    builder.addTextBody("profile", jsonProfile, ContentType.APPLICATION_JSON);

    HttpEntity entity = builder.build();

    // Create a post method instance.
    HttpPost postMethod = new HttpPost(postUrl);


    // Provide custom retry handler is necessary
    /*postMethod.getParams().setParameter(HttpMethodParams.RETRY_HANDLER,
        new DefaultHttpMethodRetryHandler(5, true));
    */
    try {
      postMethod.setEntity(entity);
      long start = System.currentTimeMillis();
      CloseableHttpResponse response = client.execute(postMethod);
      //postMethod.setRequestEntity(new MultipartRequestEntity(parts, postMethod.getParams()));


      int statusCode = response.getCode();
      // Distinguish success vs failure
      if (statusCode >= 200 && statusCode < 300) {
        successCount.incrementAndGet();
      } else {
        failCount.incrementAndGet();
        System.err.println("Post Method failed: " + statusCode);
      }

      long end = System.currentTimeMillis();

      long latency = end - start;
      data.add(RowFactory.create(start, "POST", latency, statusCode));

      // Read the response body.
      //byte[] responseBody = response.getEntity().getContent().readAllBytes();
      //System.out.println(new String(responseBody));

      // Consume response content
      EntityUtils.consume(response.getEntity());
      long endtime = System.currentTimeMillis();
      System.out.println("POST END: " + endtime + Thread.currentThread().getName());
    } catch (IOException e) {
      System.err.println("Fatal transport error: " + e.getMessage());
      e.printStackTrace();
    }
  }

  public static int getSuccessCount() {
    return successCount.get();
  }

  public static int getFailCount() {
    return failCount.get();
  }
}
