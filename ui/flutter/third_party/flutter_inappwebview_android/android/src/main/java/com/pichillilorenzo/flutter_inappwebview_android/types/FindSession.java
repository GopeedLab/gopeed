package com.pichillilorenzo.flutter_inappwebview_android.types;

import java.util.HashMap;
import java.util.Map;

public class FindSession {
  private int resultCount;
  private int highlightedResultIndex;
  private int searchResultDisplayStyle = 2; // matches NONE of iOS

  public FindSession(int resultCount, int highlightedResultIndex) {
    this.resultCount = resultCount;
    this.highlightedResultIndex = highlightedResultIndex;
  }

  public Map<String, Object> toMap() {
    Map<String, Object> obj = new HashMap<>();
    obj.put("resultCount", resultCount);
    obj.put("highlightedResultIndex", highlightedResultIndex);
    obj.put("searchResultDisplayStyle", searchResultDisplayStyle);
    return obj;
  }

  public int getResultCount() {
    return resultCount;
  }

  public void setResultCount(int resultCount) {
    this.resultCount = resultCount;
  }

  public int getHighlightedResultIndex() {
    return highlightedResultIndex;
  }

  public void setHighlightedResultIndex(int highlightedResultIndex) {
    this.highlightedResultIndex = highlightedResultIndex;
  }

  public int getSearchResultDisplayStyle() {
    return searchResultDisplayStyle;
  }

  public void setSearchResultDisplayStyle(int searchResultDisplayStyle) {
    this.searchResultDisplayStyle = searchResultDisplayStyle;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    FindSession that = (FindSession) o;

    if (resultCount != that.resultCount) return false;
    if (highlightedResultIndex != that.highlightedResultIndex) return false;
    return searchResultDisplayStyle == that.searchResultDisplayStyle;
  }

  @Override
  public int hashCode() {
    int result = resultCount;
    result = 31 * result + highlightedResultIndex;
    result = 31 * result + searchResultDisplayStyle;
    return result;
  }

  @Override
  public String toString() {
    return "FindSession{" +
            "resultCount=" + resultCount +
            ", highlightedResultIndex=" + highlightedResultIndex +
            ", searchResultDisplayStyle=" + searchResultDisplayStyle +
            '}';
  }
}
