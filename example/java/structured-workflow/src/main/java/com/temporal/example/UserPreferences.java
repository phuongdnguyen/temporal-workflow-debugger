package com.temporal.example;

/**
 * User preferences for the onboarding workflow.
 */
public class UserPreferences {
    private final String theme;
    private final String language;
    private final boolean notifications;
    private final String timezone;
    
    public UserPreferences(String theme, String language, boolean notifications, String timezone) {
        this.theme = theme;
        this.language = language;
        this.notifications = notifications;
        this.timezone = timezone;
    }
    
    public String getTheme() {
        return theme;
    }
    
    public String getLanguage() {
        return language;
    }
    
    public boolean isNotifications() {
        return notifications;
    }
    
    public String getTimezone() {
        return timezone;
    }
    
    @Override
    public String toString() {
        return "UserPreferences{" +
                "theme='" + theme + '\'' +
                ", language='" + language + '\'' +
                ", notifications=" + notifications +
                ", timezone='" + timezone + '\'' +
                '}';
    }
}
