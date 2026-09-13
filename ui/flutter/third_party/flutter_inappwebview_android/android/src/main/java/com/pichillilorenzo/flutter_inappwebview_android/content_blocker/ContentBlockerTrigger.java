package com.pichillilorenzo.flutter_inappwebview_android.content_blocker;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.Map;
import java.util.regex.Pattern;

public class ContentBlockerTrigger {

    @NonNull
    private String urlFilter;
    private Pattern urlFilterPatternCompiled;
    private Boolean urlFilterIsCaseSensitive;
    private List<ContentBlockerTriggerResourceType> resourceType = new ArrayList<>();
    private List<String> ifDomain = new ArrayList<>();
    private List<String> unlessDomain = new ArrayList<>();
    private List<String> loadType = new ArrayList<>();
    private List<String> ifTopUrl = new ArrayList<>();
    private List<String> unlessTopUrl = new ArrayList<>();

    public ContentBlockerTrigger(@NonNull String urlFilter, @Nullable Boolean urlFilterIsCaseSensitive, @Nullable List<ContentBlockerTriggerResourceType> resourceType,
                                 @Nullable List<String> ifDomain, @Nullable List<String> unlessDomain, @Nullable List<String> loadType,
                                 @Nullable List<String> ifTopUrl, @Nullable List<String> unlessTopUrl) {
        this.urlFilterIsCaseSensitive = urlFilterIsCaseSensitive != null ? urlFilterIsCaseSensitive : false;

        this.urlFilter = urlFilter;
        this.urlFilterPatternCompiled = Pattern.compile(this.urlFilter, this.urlFilterIsCaseSensitive ? 0 : Pattern.CASE_INSENSITIVE);

        this.resourceType = resourceType != null ? resourceType : this.resourceType;
        this.ifDomain = ifDomain != null ? ifDomain : this.ifDomain;
        this.unlessDomain = unlessDomain != null ? unlessDomain : this.unlessDomain;
        if ((!(this.ifDomain.isEmpty() || this.unlessDomain.isEmpty()) != false))
            throw new AssertionError();
        this.loadType = loadType != null ? loadType : this.loadType;
        if ((this.loadType.size() > 2)) throw new AssertionError();
        this.ifTopUrl = ifTopUrl != null ? ifTopUrl : this.ifTopUrl;
        this.unlessTopUrl = unlessTopUrl != null ? unlessTopUrl : this.unlessTopUrl;
        if ((!(this.ifTopUrl.isEmpty() || this.unlessTopUrl.isEmpty()) != false))
            throw new AssertionError();
    }

    public static ContentBlockerTrigger fromMap(Map<String, Object> map) {
        String urlFilter = (String) map.get("url-filter");
        Boolean urlFilterIsCaseSensitive = (Boolean) map.get("url-filter-is-case-sensitive");
        List<String> resourceTypeStringList = (List<String>) map.get("resource-type");
        List<ContentBlockerTriggerResourceType> resourceType = new ArrayList<>();
        if (resourceTypeStringList != null) {
            for (String type : resourceTypeStringList) {
                resourceType.add(ContentBlockerTriggerResourceType.fromValue(type));
            }
        } else {
            resourceType.addAll(Arrays.asList(ContentBlockerTriggerResourceType.values()));
        }
        List<String> ifDomain = (List<String>) map.get("if-domain");
        List<String> unlessDomain = (List<String>) map.get("unless-domain");
        List<String> loadType = (List<String>) map.get("load-type");
        List<String> ifTopUrl = (List<String>) map.get("if-top-url");
        List<String> unlessTopUrl = (List<String>) map.get("unless-top-url");
        return new ContentBlockerTrigger(urlFilter, urlFilterIsCaseSensitive, resourceType, ifDomain, unlessDomain, loadType, ifTopUrl, unlessTopUrl);
    }

    @NonNull
    public String getUrlFilter() {
        return urlFilter;
    }

    public void setUrlFilter(@NonNull String urlFilter) {
        this.urlFilter = urlFilter;
    }

    public Pattern getUrlFilterPatternCompiled() {
        return urlFilterPatternCompiled;
    }

    public void setUrlFilterPatternCompiled(Pattern urlFilterPatternCompiled) {
        this.urlFilterPatternCompiled = urlFilterPatternCompiled;
    }

    public Boolean getUrlFilterIsCaseSensitive() {
        return urlFilterIsCaseSensitive;
    }

    public void setUrlFilterIsCaseSensitive(Boolean urlFilterIsCaseSensitive) {
        this.urlFilterIsCaseSensitive = urlFilterIsCaseSensitive;
    }

    public List<ContentBlockerTriggerResourceType> getResourceType() {
        return resourceType;
    }

    public void setResourceType(List<ContentBlockerTriggerResourceType> resourceType) {
        this.resourceType = resourceType;
    }

    public List<String> getIfDomain() {
        return ifDomain;
    }

    public void setIfDomain(List<String> ifDomain) {
        this.ifDomain = ifDomain;
    }

    public List<String> getUnlessDomain() {
        return unlessDomain;
    }

    public void setUnlessDomain(List<String> unlessDomain) {
        this.unlessDomain = unlessDomain;
    }

    public List<String> getLoadType() {
        return loadType;
    }

    public void setLoadType(List<String> loadType) {
        this.loadType = loadType;
    }

    public List<String> getIfTopUrl() {
        return ifTopUrl;
    }

    public void setIfTopUrl(List<String> ifTopUrl) {
        this.ifTopUrl = ifTopUrl;
    }

    public List<String> getUnlessTopUrl() {
        return unlessTopUrl;
    }

    public void setUnlessTopUrl(List<String> unlessTopUrl) {
        this.unlessTopUrl = unlessTopUrl;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;

        ContentBlockerTrigger that = (ContentBlockerTrigger) o;

        if (!urlFilter.equals(that.urlFilter)) return false;
        if (!urlFilterPatternCompiled.equals(that.urlFilterPatternCompiled)) return false;
        if (!urlFilterIsCaseSensitive.equals(that.urlFilterIsCaseSensitive)) return false;
        if (!resourceType.equals(that.resourceType)) return false;
        if (!ifDomain.equals(that.ifDomain)) return false;
        if (!unlessDomain.equals(that.unlessDomain)) return false;
        if (!loadType.equals(that.loadType)) return false;
        if (!ifTopUrl.equals(that.ifTopUrl)) return false;
        return unlessTopUrl.equals(that.unlessTopUrl);
    }

    @Override
    public int hashCode() {
        int result = urlFilter.hashCode();
        result = 31 * result + urlFilterPatternCompiled.hashCode();
        result = 31 * result + urlFilterIsCaseSensitive.hashCode();
        result = 31 * result + resourceType.hashCode();
        result = 31 * result + ifDomain.hashCode();
        result = 31 * result + unlessDomain.hashCode();
        result = 31 * result + loadType.hashCode();
        result = 31 * result + ifTopUrl.hashCode();
        result = 31 * result + unlessTopUrl.hashCode();
        return result;
    }

    @Override
    public String toString() {
        return "ContentBlockerTrigger{" +
                "urlFilter='" + urlFilter + '\'' +
                ", urlFilterPatternCompiled=" + urlFilterPatternCompiled +
                ", urlFilterIsCaseSensitive=" + urlFilterIsCaseSensitive +
                ", resourceType=" + resourceType +
                ", ifDomain=" + ifDomain +
                ", unlessDomain=" + unlessDomain +
                ", loadType=" + loadType +
                ", ifTopUrl=" + ifTopUrl +
                ", unlessTopUrl=" + unlessTopUrl +
                '}';
    }
}
