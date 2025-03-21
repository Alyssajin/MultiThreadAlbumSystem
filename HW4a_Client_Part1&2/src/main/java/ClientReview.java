import org.apache.hadoop.shaded.org.apache.http.Consts;
import org.apache.hc.client5.http.classic.methods.HttpPost;
import org.apache.hc.client5.http.entity.EntityBuilder;
import org.apache.hc.client5.http.entity.UrlEncodedFormEntity;
import org.apache.hc.client5.http.impl.classic.CloseableHttpClient;
import org.apache.hc.client5.http.impl.classic.CloseableHttpResponse;
import org.apache.hc.core5.http.NameValuePair;
import org.apache.hc.core5.http.io.entity.EntityUtils;
import org.apache.hc.core5.http.message.BasicNameValuePair;
import org.apache.hc.core5.net.URIBuilder;
import org.apache.spark.sql.Row;

import java.io.IOException;
import java.net.URI;
import java.net.URISyntaxException;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.atomic.AtomicInteger;

public class ClientReview implements Runnable{
  private static AtomicInteger successCount = new AtomicInteger(0);
  private static AtomicInteger failCount = new AtomicInteger(0);
  private final String IPAddr;
  private final CloseableHttpClient client;
  private final List<Row> data;
  private final Like like;
  private final String albumId;

  public ClientReview(String IPAddr, CloseableHttpClient client, List<Row> data,
                      Like like, String albumId) {
    this.client = client;
    this.albumId = albumId;
    this.IPAddr = IPAddr;
    this.data = data;
    this.like = like;
  }

  @Override
  public void run() {
    try {

      URI uri = new URIBuilder()
              .setScheme("http")
              .setHost(IPAddr)
              .setPath("/" + like + "/" + albumId)
              .build();

      List<NameValuePair> params = new ArrayList<NameValuePair>(2);
      params.add(new BasicNameValuePair("albumId", albumId));
      params.add(new BasicNameValuePair("like", like.toString()));

      HttpPost postMethod = new HttpPost(uri);
      postMethod.setEntity(new UrlEncodedFormEntity(params, Consts.UTF_8));
      System.out.println(params);

      CloseableHttpResponse response = client.execute(postMethod);
      if (response.getCode() >= 200 && response.getCode() < 300) {
        successCount.incrementAndGet();
      } else {
        failCount.incrementAndGet();
      }

      EntityUtils.consume(response.getEntity());

    } catch (URISyntaxException e) {
      throw new RuntimeException(e);
    } catch (IOException e) {
      throw new RuntimeException(e);
    }

  }
}
