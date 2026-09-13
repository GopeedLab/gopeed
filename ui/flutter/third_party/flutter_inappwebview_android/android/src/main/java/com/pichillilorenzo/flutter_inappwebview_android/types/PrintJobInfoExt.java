package com.pichillilorenzo.flutter_inappwebview_android.types;

import android.os.Build;
import android.print.PrintJobInfo;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.annotation.RequiresApi;

import java.util.HashMap;
import java.util.Map;

@RequiresApi(api = Build.VERSION_CODES.KITKAT)
public class PrintJobInfoExt {
  private int state;
  private int copies;
  @Nullable
  private Integer numberOfPages;
  private long creationTime;
  @NonNull
  private String label;
  @Nullable
  private String printerId;
  @Nullable
  private PrintAttributesExt attributes;
  
  @Nullable
  public static PrintJobInfoExt fromPrintJobInfo(@Nullable PrintJobInfo info) {
    if (info == null) {
      return null;
    }
    PrintJobInfoExt printJobInfoExt = new PrintJobInfoExt();
    printJobInfoExt.state = info.getState();
    printJobInfoExt.copies = info.getCopies();
    printJobInfoExt.numberOfPages = info.getPages() != null ? info.getPages().length : null;
    printJobInfoExt.creationTime = info.getCreationTime();
    printJobInfoExt.label = info.getLabel();
    printJobInfoExt.printerId = info.getPrinterId() != null ? info.getPrinterId().getLocalId() : null;
    printJobInfoExt.attributes = PrintAttributesExt.fromPrintAttributes(info.getAttributes());
    return printJobInfoExt;
  }

  public Map<String, Object> toMap() {
    Map<String, Object> obj = new HashMap<>();
    obj.put("state", state);
    obj.put("copies", copies);
    obj.put("numberOfPages", numberOfPages);
    obj.put("creationTime", creationTime);
    obj.put("label", label);
    Map<String, Object> printer = new HashMap<>();
    printer.put("id", printerId);
    obj.put("printer", printer);
    obj.put("attributes", attributes != null ? attributes.toMap() : null);
    return obj;
  }
}
