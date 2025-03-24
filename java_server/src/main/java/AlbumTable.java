import java.sql.Connection;
import java.sql.SQLException;
import java.sql.Statement;

public class AlbumTable {
  private String dbName;
  private Connection con;
  private String dbms;

  public AlbumTable(Connection connArg, String dbNameArg, String dbmsArg) {
    this.con = connArg;
    this.dbName = dbNameArg;
    this.dbms = dbmsArg;
  }

  /**
   * Create ALBUM table and PROFILE table.
   * ALBUM table has columns ALBUM_ID, TITLE, ARTIST, LIKE, YEAR, IMAGE, IMAGE_SIZE.
   * ALBUM_ID is the primary key of ALBUM table.
   * @throws SQLException if a database access error occurs
   */
  public void createTable() throws SQLException {
    String createAlbumString = "create table ALBUM " + "(ALBUM_ID INT NOT NULL, "
        + "TITLE VARCHAR(40) NOT NULL, " + "ARTIST VARCHAR(40) NOT NULL, " + "LIKE BOOLEAN DEFAULT FALSE, "
        + "YEAR INT NOT NULL, " + "IMAGE BLOB NOT NULL, " + "IMAGE_SIZE INT NOT NULL, " + "PRIMARY KEY (ALBUM_ID))";
    try {
      con.createStatement().executeUpdate(createAlbumString);
    } catch (SQLException e) {
      System.out.println("Table already exists");
    }
  }

  public void insertRow(String title, String artist, boolean like, int year, byte[] image, int imageSize) {
    try (Statement stmt = con.createStatement()){

    } catch (SQLException e) {
      System.out.println("Insert failed");
    }

  }


}
