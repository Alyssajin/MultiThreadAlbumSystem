import com.google.gson.Gson;

import java.nio.charset.StandardCharsets;
import javax.servlet.*;
import javax.servlet.http.*;
import javax.servlet.annotation.*;
import java.io.IOException;

@MultipartConfig
public class Server extends HttpServlet {
  @Override
  protected void doGet(HttpServletRequest request, HttpServletResponse response)
      throws ServletException, IOException {
    response.setContentType("application/json");
    String url = request.getPathInfo();
    if (url == null || url.isEmpty()) {
      response.setStatus(HttpServletResponse.SC_NOT_FOUND);
      response.getWriter().write("missing parameters");
      return;
    }
    String[] urlParts = url.split("/");
    String lastPart = urlParts[urlParts.length - 1];
    String newUrl = url.substring(0, url.length() - lastPart.length() - 1);
    //System.out.println(newUrl);
    //System.out.println(lastPart);
    if (newUrl.equals("/") && lastPart.equals("1")) {
      response.setStatus(HttpServletResponse.SC_OK);
      response.setContentType("application/json");
      AlbumTable album = new AlbumTable();
      String respondAlbum = new Gson().toJson(album);
      response.getWriter().write(respondAlbum);
    } else {
      response.setStatus(HttpServletResponse.SC_NOT_FOUND);
      response.getWriter().write("invalid parameters");
    }
  }

  @Override
  protected void doPost(HttpServletRequest request, HttpServletResponse response)
      throws ServletException, IOException {
    response.setContentType("text/plain");
    String url = request.getPathInfo();
    if (url == null || url.isEmpty()) {
      response.setStatus(HttpServletResponse.SC_NOT_FOUND);
      response.getWriter().write("missing parameters");
      return;
    }
    if (!url.equals("/album")) {
      response.setStatus(HttpServletResponse.SC_NOT_FOUND);
      response.getWriter().write("invalid parameters");
      return;
    }
    String artist = "";
    String title = "";
    String year = "";
    for (Part p : request.getParts()) {
      if (p.getName().equals("image")) {
        //InputStream s = p.getInputStream();
        //byte[] data = s.readAllBytes();
        response.getWriter().write(String.valueOf(25303));
        //s.close();
      }
      if (p.getName().equals("profile[artist]")) {
        artist = new String(p.getInputStream().readAllBytes(), StandardCharsets.UTF_8);;
      }
      if (p.getName().equals("profile[title]")) {
        title = new String(p.getInputStream().readAllBytes(), StandardCharsets.UTF_8);;
      }
      if (p.getName().equals("profile[year]")) {
        year = new String(p.getInputStream().readAllBytes(), StandardCharsets.UTF_8);;
      }
    }
    AlbumTable newAlbum = new AlbumTable(artist, title, year);
    response.setStatus(HttpServletResponse.SC_CREATED);
    response.getWriter().write(new Gson().toJson(newAlbum));
  }


}

