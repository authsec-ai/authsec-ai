import React, {createContext, useContext, useState, useEffect, ReactNode} from 'react';
import {useColorScheme} from 'react-native';
import AsyncStorage from '@react-native-async-storage/async-storage';

type ThemeMode = 'light' | 'dark' | 'system';

interface ThemeColors {
  // Base
  background: string;
  surface: string;
  surfaceElevated: string;
  card: string;
  
  // Text
  text: string;
  textSecondary: string;
  textMuted: string;
  textInverse: string;
  
  // Brand
  primary: string;
  primaryLight: string;
  primaryDark: string;
  
  // Semantic
  success: string;
  successLight: string;
  error: string;
  errorLight: string;
  warning: string;
  warningLight: string;
  info: string;
  infoLight: string;
  
  // Border & Divider
  border: string;
  divider: string;
  
  // Input
  inputBackground: string;
  inputBorder: string;
  inputPlaceholder: string;
  
  // Shadow
  shadow: string;
  
  // Overlay
  overlay: string;
}

const lightColors: ThemeColors = {
  // Base - Clean whites like Claude
  background: '#FFFFFF',
  surface: '#FFFFFF',
  surfaceElevated: '#FAFAFA',
  card: '#FFFFFF',
  
  // Text - High contrast
  text: '#191919',
  textSecondary: '#666666',
  textMuted: '#999999',
  textInverse: '#FFFFFF',
  
  // Brand - Black for light theme
  primary: '#000000',
  primaryLight: '#F5F5F5',
  primaryDark: '#1A1A1A',
  
  // Semantic
  success: '#00C853',
  successLight: '#E8F5E9',
  error: '#FF3B30',
  errorLight: '#FFEBEE',
  warning: '#FF9500',
  warningLight: '#FFF3E0',
  info: '#007AFF',
  infoLight: '#E3F2FD',
  
  // Border & Divider
  border: '#EFEFEF',
  divider: '#F5F5F5',
  
  // Input
  inputBackground: '#FAFAFA',
  inputBorder: '#DBDBDB',
  inputPlaceholder: '#999999',
  
  // Shadow
  shadow: '#000000',
  
  // Overlay
  overlay: 'rgba(0, 0, 0, 0.5)',
};

const darkColors: ThemeColors = {
  // Base - Deep blacks like Instagram dark mode
  background: '#000000',
  surface: '#121212',
  surfaceElevated: '#1C1C1E',
  card: '#1C1C1E',
  
  // Text - Clean whites
  text: '#FFFFFF',
  textSecondary: '#A0A0A0',
  textMuted: '#666666',
  textInverse: '#000000',
  
  // Brand - White for dark theme
  primary: '#FFFFFF',
  primaryLight: '#1A1A1A',
  primaryDark: '#E5E5E5',
  
  // Semantic
  success: '#30D158',
  successLight: '#0D2818',
  error: '#FF453A',
  errorLight: '#2C1215',
  warning: '#FFD60A',
  warningLight: '#2C2508',
  info: '#0A84FF',
  infoLight: '#0A1929',
  
  // Border & Divider
  border: '#2C2C2E',
  divider: '#1C1C1E',
  
  // Input
  inputBackground: '#1C1C1E',
  inputBorder: '#3A3A3C',
  inputPlaceholder: '#666666',
  
  // Shadow
  shadow: '#000000',
  
  // Overlay
  overlay: 'rgba(0, 0, 0, 0.85)',
};

interface ThemeContextType {
  colors: ThemeColors;
  isDark: boolean;
  themeMode: ThemeMode;
  setThemeMode: (mode: ThemeMode) => void;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

const THEME_STORAGE_KEY = '@authsec_theme_mode';

export const ThemeProvider: React.FC<{children: ReactNode}> = ({children}) => {
  const systemColorScheme = useColorScheme();
  const [themeMode, setThemeModeState] = useState<ThemeMode>('system');
  const [isLoaded, setIsLoaded] = useState(false);

  useEffect(() => {
    loadThemePreference();
  }, []);

  const loadThemePreference = async () => {
    try {
      const stored = await AsyncStorage.getItem(THEME_STORAGE_KEY);
      if (stored && ['light', 'dark', 'system'].includes(stored)) {
        setThemeModeState(stored as ThemeMode);
      }
    } catch (error) {
      console.error('Failed to load theme preference:', error);
    } finally {
      setIsLoaded(true);
    }
  };

  const setThemeMode = async (mode: ThemeMode) => {
    setThemeModeState(mode);
    try {
      await AsyncStorage.setItem(THEME_STORAGE_KEY, mode);
    } catch (error) {
      console.error('Failed to save theme preference:', error);
    }
  };

  const isDark = themeMode === 'system' 
    ? systemColorScheme === 'dark' 
    : themeMode === 'dark';

  const colors = isDark ? darkColors : lightColors;

  // Don't block rendering while loading - let the app render with default theme
  // The theme will update once loaded
  return (
    <ThemeContext.Provider value={{colors, isDark, themeMode, setThemeMode}}>
      {children}
    </ThemeContext.Provider>
  );
};

export const useTheme = () => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};

export type {ThemeColors, ThemeMode};
