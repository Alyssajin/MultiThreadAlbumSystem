import org.apache.hc.client5.http.classic.methods.HttpPost;
import org.apache.hc.client5.http.impl.classic.CloseableHttpClient;
import org.apache.hc.client5.http.impl.classic.CloseableHttpResponse;
import org.apache.hc.core5.http.io.entity.EntityUtils;
import org.apache.hc.core5.net.URIBuilder;
import org.apache.spark.sql.Row;
import org.apache.spark.sql.RowFactory;

import java.io.IOException;
import java.net.URI;
import java.net.URISyntaxException;
import java.util.List;
import java.util.concurrent.atomic.AtomicInteger;

public class ClientReview implements Runnable{
  private static AtomicInteger successCount = new AtomicInteger(0);
  private static AtomicInteger failCount = new AtomicInteger(0);
  private final String IPAddr;
  private final CloseableHttpClient client;
  private final List<Row> data;
  private final Like likeOrNot;
  private final String albumId;
  private final int port;

  public ClientReview(String IPAddr, CloseableHttpClient client, List<Row> data,
                      Like likeOrNot, String albumId, int port) {
    this.client = client;
    this.albumId = albumId;
    this.IPAddr = IPAddr;
    this.data = data;
    this.likeOrNot = likeOrNot;
    this.port = port;
  }

  @Override
  public void run() {
    try {

      URI uri = new URIBuilder()
              .setScheme("http")
              .setHost(IPAddr)
              .setPort(port)
              .setPath("/album/" + likeOrNot + "/" + albumId)
              .build();

      HttpPost postMethod = new HttpPost(uri);
//      long start = System.currentTimeMillis();

      CloseableHttpResponse response = client.execute(postMethod);
      if (response.getCode() >= 200 && response.getCode() < 300) {
        successCount.incrementAndGet();
      } else {
        failCount.incrementAndGet();
      }

//      long latency = System.currentTimeMillis() - start;
//      int statusCode = response.getCode();
//      data.add(RowFactory.create(start, "POST", latency, statusCode));

      EntityUtils.consume(response.getEntity());

    } catch (URISyntaxException e) {
      throw new RuntimeException(e);
    } catch (IOException e) {
      throw new RuntimeException(e);
    }

  }
}
