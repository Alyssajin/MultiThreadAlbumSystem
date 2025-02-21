import java.io.IOException;
import java.util.List;

import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.Future;
import org.apache.hc.client5.http.async.methods.SimpleHttpRequest;
import org.apache.hc.client5.http.async.methods.SimpleHttpResponse;
import org.apache.hc.client5.http.async.methods.SimpleRequestBuilder;
import org.apache.hc.client5.http.classic.methods.HttpGet;
import org.apache.hc.client5.http.impl.async.CloseableHttpAsyncClient;
import org.apache.hc.client5.http.impl.classic.CloseableHttpClient;
import org.apache.hc.client5.http.impl.classic.CloseableHttpResponse;
import org.apache.hc.core5.http.HttpEntity;
import org.apache.hc.core5.http.HttpStatus;
import org.apache.hc.core5.http.io.entity.EntityUtils;
import org.apache.spark.sql.Row;
import org.apache.spark.sql.RowFactory;

public class ClientGet implements Runnable {
  private String getUrl;
  private CloseableHttpClient client;

  public ClientGet(String IPAddr, CloseableHttpClient client, List<Row> data) {
    this.getUrl = "http://" + IPAddr + "/IGORTON/AlbumStore/1.0.0/albums/1";
    this.client = client;
  }

  // stolen from https://hc.apache.org/httpclient-legacy/tutorial.html
  public void run() {

    // Create a method instance.
    HttpGet getMethod = new HttpGet(getUrl);

    // Create a post method instance.

    // Provide custom retry handler is necessary
    /*
    getMethod.getParams().setParameter(HttpMethodParams.RETRY_HANDLER,
        new DefaultHttpMethodRetryHandler(5, false));
     */
    try {
      // Execute the method.
      CloseableHttpResponse response = client.execute(getMethod);

      int statusCode = response.getCode();
      if (statusCode != HttpStatus.SC_OK) {
        System.err.println("Method failed: " + statusCode);
      }

      // Read the response body.
      byte[] responseBody = response.getEntity().getContent().readAllBytes();
      //System.out.println(new String(responseBody));

    } catch (IOException e) {
      throw new RuntimeException(e);
    }
  }
}
