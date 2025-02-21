import java.io.File;
import java.io.IOException;
import java.util.List;
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
  private String postUrl;
  private CloseableHttpClient client;
  private List<Row> data;
  private File file;


  public ClientPost(String IPAddr, CloseableHttpClient client, List<Row> data, File file) {
    this.postUrl = "http://" + IPAddr + "/IGORTON/AlbumStore/1.0.0/albums";
    this.client = client;
    this.data = data;
    this.file = file;
  }

  // stolen from https://hc.apache.org/httpclient-legacy/tutorial.html
  public void run() {
    MultipartEntityBuilder builder = MultipartEntityBuilder.create();
    builder.setMode(HttpMultipartMode.STRICT);

    builder.addBinaryBody("image",
        file, ContentType.IMAGE_JPEG,
        "Example.jpg");
    builder.addTextBody("profile[artist]", "1234", ContentType.TEXT_PLAIN);
    builder.addTextBody("profile[title]", "2345", ContentType.TEXT_PLAIN);
    builder.addTextBody("profile[year]", "3456", ContentType.TEXT_PLAIN);
    HttpEntity entity = builder.build();

    // Create a post method instance.
    HttpPost postMethod = new HttpPost(postUrl);


    // Provide custom retry handler is necessary
    /*postMethod.getParams().setParameter(HttpMethodParams.RETRY_HANDLER,
        new DefaultHttpMethodRetryHandler(5, true));
    */
    try {
      /*
      Part[] parts = {new FilePart("image",
          new File("/Users/boyuansun/Desktop/25Spring/CS 6650/Example.jpg")),
          new StringPart("profile[artist]", "1234"),
          new StringPart("profile[title]", "2345"),
          new StringPart("profile[year]", "3456")
      };
      */
      postMethod.setEntity(entity);
      long start = System.currentTimeMillis();
      CloseableHttpResponse response = client.execute(postMethod);
      //postMethod.setRequestEntity(new MultipartRequestEntity(parts, postMethod.getParams()));


      int statusCode = response.getCode();
      //System.out.println(statusCode);
      if (statusCode != HttpStatus.SC_CREATED) {
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

    } catch (IOException e) {
      System.err.println("Fatal transport error: " + e.getMessage());
      e.printStackTrace();
    }
  }
}
