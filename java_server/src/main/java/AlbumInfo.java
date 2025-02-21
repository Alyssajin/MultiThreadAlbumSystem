public class AlbumInfo {
  private String artist;
  private String title;
  private String year;

  private void setArtist(String artist) {
    this.artist = artist;
  }

  private void setTitle(String title) {
    this.title = title;
  }

  private void setYear(String year) {
    this.year = year;
  }

  public AlbumInfo(String artist, String title, String year) {
    this.artist = artist;
    this.title = title;
    this.year = year;
  }

  public AlbumInfo() {
    this.artist = "SixPistols";
    this.title = "NeverMindTheBollocks";
    this.year = "1977";
  }
}
