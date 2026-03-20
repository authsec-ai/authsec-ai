import React, {useState, useCallback} from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  RefreshControl,
} from 'react-native';
import {useTheme} from '../context/ThemeContext';
import {Ionicons} from '@expo/vector-icons';
import {useFocusEffect} from '@react-navigation/native';
import {getActivityLogs, clearActivityLogs, ActivityLog} from '../services/storage';
import {Alert} from 'react-native';

const ActivityScreen = () => {
  const {colors, isDark} = useTheme();
  const [activities, setActivities] = useState<ActivityLog[]>([]);
  const [refreshing, setRefreshing] = useState(false);

  const loadActivities = async () => {
    const logs = await getActivityLogs();
    setActivities(logs);
  };

  useFocusEffect(
    useCallback(() => {
      loadActivities();
    }, []),
  );

  const onRefresh = async () => {
    setRefreshing(true);
    await loadActivities();
    setRefreshing(false);
  };

  const handleClearAll = () => {
    Alert.alert(
      'Clear All Activity',
      'Are you sure you want to clear all activity logs?',
      [
        {text: 'Cancel', style: 'cancel'},
        {
          text: 'Clear',
          style: 'destructive',
          onPress: async () => {
            await clearActivityLogs();
            setActivities([]);
          },
        },
      ],
    );
  };

  const getActivityIcon = (type: ActivityLog['type']) => {
    switch (type) {
      case 'auth_approved':
        return 'checkmark-circle';
      case 'auth_denied':
        return 'close-circle';
      case 'totp_added':
        return 'add-circle';
      case 'totp_deleted':
        return 'trash';
      default:
        return 'information-circle';
    }
  };

  const getActivityColor = (type: ActivityLog['type']) => {
    switch (type) {
      case 'auth_approved':
        return colors.success;
      case 'auth_denied':
        return colors.error;
      case 'totp_added':
        return colors.primary;
      case 'totp_deleted':
        return colors.textMuted;
      default:
        return colors.info;
    }
  };

  const formatTimestamp = (timestamp: number) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;

    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: date.getFullYear() !== now.getFullYear() ? 'numeric' : undefined,
    });
  };

  const styles = createStyles(colors, isDark);

  return (
    <View style={styles.container}>
      {/* Header */}
      <View style={styles.header}>
        <Text style={styles.headerTitle}>Activity</Text>
        {activities.length > 0 && (
          <TouchableOpacity onPress={handleClearAll}>
            <Text style={styles.clearButton}>Clear All</Text>
          </TouchableOpacity>
        )}
      </View>

      <ScrollView
        style={styles.scrollView}
        contentContainerStyle={styles.scrollContent}
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            onRefresh={onRefresh}
            tintColor={colors.primary}
          />
        }>
        {activities.length === 0 ? (
          <View style={styles.emptyState}>
            <Ionicons name="time-outline" size={64} color={colors.textMuted} />
            <Text style={styles.emptyTitle}>No Activity Yet</Text>
            <Text style={styles.emptyDescription}>
              Your activity history will appear here
            </Text>
          </View>
        ) : (
          <View style={styles.activityList}>
            {activities.map((activity) => (
              <View key={activity.id} style={styles.activityCard}>
                <View style={styles.activityIcon}>
                  <Ionicons
                    name={getActivityIcon(activity.type)}
                    size={24}
                    color={getActivityColor(activity.type)}
                  />
                </View>
                <View style={styles.activityContent}>
                  <Text style={styles.activityTitle}>{activity.title}</Text>
                  <Text style={styles.activityDescription}>
                    {activity.description}
                  </Text>
                  <Text style={styles.activityTime}>
                    {formatTimestamp(activity.timestamp)}
                  </Text>
                </View>
              </View>
            ))}
          </View>
        )}
      </ScrollView>
    </View>
  );
};

const createStyles = (colors: any, isDark: boolean) =>
  StyleSheet.create({
    container: {
      flex: 1,
      backgroundColor: colors.background,
    },
    header: {
      flexDirection: 'row',
      justifyContent: 'space-between',
      alignItems: 'center',
      padding: 20,
      paddingTop: 60,
      backgroundColor: colors.card,
      borderBottomWidth: 1,
      borderBottomColor: colors.border,
    },
    headerTitle: {
      fontSize: 28,
      fontWeight: '700',
      color: colors.text,
    },
    clearButton: {
      fontSize: 14,
      fontWeight: '600',
      color: colors.error,
    },
    scrollView: {
      flex: 1,
    },
    scrollContent: {
      padding: 20,
    },
    emptyState: {
      alignItems: 'center',
      justifyContent: 'center',
      paddingTop: 100,
    },
    emptyTitle: {
      fontSize: 20,
      fontWeight: '600',
      color: colors.text,
      marginTop: 16,
    },
    emptyDescription: {
      fontSize: 14,
      color: colors.textMuted,
      marginTop: 8,
      textAlign: 'center',
    },
    activityList: {
      gap: 12,
    },
    activityCard: {
      flexDirection: 'row',
      padding: 16,
      backgroundColor: colors.card,
      borderRadius: 16,
      shadowColor: colors.shadow,
      shadowOffset: {width: 0, height: 2},
      shadowOpacity: isDark ? 0.3 : 0.08,
      shadowRadius: 8,
      elevation: 3,
    },
    activityIcon: {
      width: 48,
      height: 48,
      borderRadius: 24,
      backgroundColor: colors.background,
      justifyContent: 'center',
      alignItems: 'center',
      marginRight: 12,
    },
    activityContent: {
      flex: 1,
    },
    activityTitle: {
      fontSize: 16,
      fontWeight: '600',
      color: colors.text,
      marginBottom: 4,
    },
    activityDescription: {
      fontSize: 14,
      color: colors.textSecondary,
      marginBottom: 6,
    },
    activityTime: {
      fontSize: 12,
      color: colors.textMuted,
    },
  });

export default ActivityScreen;
