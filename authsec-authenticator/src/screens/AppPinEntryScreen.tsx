import React, {useState, useRef, useEffect} from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  Alert,
  Animated,
  StatusBar,
  ActivityIndicator,
  BackHandler,
  Platform,
} from 'react-native';
import {useTheme} from '../context/ThemeContext';
import {getAppPin, getBiometricEnabled} from '../services/storage';
import {authenticateWithBiometric, getBiometricType} from '../services/biometric';
import {Ionicons} from '@expo/vector-icons';

const AppPinEntryScreen = ({navigation}: any) => {
  const {colors, isDark} = useTheme();
  const [pin, setPin] = useState('');
  const [loading, setLoading] = useState(false);
  const [attempts, setAttempts] = useState(0);
  const [biometricEnabled, setBiometricEnabled] = useState(false);
  const [biometricType, setBiometricType] = useState('Biometric');

  const fadeAnim = useRef(new Animated.Value(0)).current;
  const shakeAnim = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    Animated.timing(fadeAnim, {
      toValue: 1,
      duration: 400,
      useNativeDriver: true,
    }).start();

    checkBiometricStatus();
    
    // Prevent back navigation on Android
    const backHandler = BackHandler.addEventListener('hardwareBackPress', () => {
      return true; // Prevent default back behavior
    });

    return () => backHandler.remove();
  }, []);

  const checkBiometricStatus = async () => {
    const enabled = await getBiometricEnabled();
    const type = await getBiometricType();
    setBiometricEnabled(enabled);
    setBiometricType(type);

    // Auto-trigger biometric if enabled
    if (enabled) {
      setTimeout(() => handleBiometricAuth(), 500);
    }
  };

  const handleBiometricAuth = async () => {
    setLoading(true);
    const success = await authenticateWithBiometric();
    setLoading(false);

    if (success) {
      navigation.replace('Home');
    }
  };

  const handleNumberPress = (num: string) => {
    if (pin.length < 6) {
      const newPin = pin + num;
      setPin(newPin);
      
      // Auto-verify when 6 digits entered
      if (newPin.length === 6) {
        verifyPin(newPin);
      }
    }
  };

  const handleDelete = () => {
    setPin(pin.slice(0, -1));
  };

  const verifyPin = async (enteredPin: string) => {
    setLoading(true);
    const storedPin = await getAppPin();
    setLoading(false);

    if (enteredPin === storedPin) {
      // Correct PIN
      navigation.replace('Home');
    } else {
      // Wrong PIN
      setAttempts(attempts + 1);
      setPin('');
      
      // Shake animation
      Animated.sequence([
        Animated.timing(shakeAnim, {
          toValue: 10,
          duration: 50,
          useNativeDriver: true,
        }),
        Animated.timing(shakeAnim, {
          toValue: -10,
          duration: 50,
          useNativeDriver: true,
        }),
        Animated.timing(shakeAnim, {
          toValue: 10,
          duration: 50,
          useNativeDriver: true,
        }),
        Animated.timing(shakeAnim, {
          toValue: 0,
          duration: 50,
          useNativeDriver: true,
        }),
      ]).start();

      if (attempts >= 4) {
        Alert.alert(
          'Too Many Attempts',
          'You have entered an incorrect PIN too many times. Please try again later.',
          [{text: 'OK'}]
        );
      } else {
        Alert.alert('Incorrect PIN', 'Please try again');
      }
    }
  };

  const renderPinDots = () => {
    return (
      <Animated.View
        style={[
          styles.dotsContainer,
          {transform: [{translateX: shakeAnim}]},
        ]}>
        {[0, 1, 2, 3, 4, 5].map((index) => (
          <View
            key={index}
            style={[
              styles.dot,
              {
                backgroundColor: pin.length > index ? colors.primary : colors.border,
                borderColor: colors.border,
              },
            ]}
          />
        ))}
      </Animated.View>
    );
  };

  const renderNumberPad = () => {
    const numbers = ['1', '2', '3', '4', '5', '6', '7', '8', '9', biometricEnabled ? 'bio' : '', '0', '⌫'];
    
    // Determine icon based on platform and biometric type
    const isAndroid = Platform.OS === 'android';
    let biometricIcon: keyof typeof Ionicons.glyphMap = 'finger-print';
    
    if (isAndroid) {
      biometricIcon = 'finger-print'; // Android fingerprint
    } else {
      if (biometricType === 'Face ID') {
        biometricIcon = 'person-circle-outline'; // Face ID
      } else {
        biometricIcon = 'finger-print'; // Touch ID
      }
    }
    
    return (
      <View style={styles.numberPad}>
        {numbers.map((num, index) => {
          if (num === '' && !biometricEnabled) {
            return <View key={index} style={styles.numberButton} />;
          }
          
          const isDelete = num === '⌫';
          const isBiometric = num === 'bio';
          
          return (
            <TouchableOpacity
              key={index}
              style={[styles.numberButton, {backgroundColor: colors.card}]}
              onPress={() => {
                if (isDelete) {
                  handleDelete();
                } else if (isBiometric) {
                  handleBiometricAuth();
                } else {
                  handleNumberPress(num);
                }
              }}
              activeOpacity={0.7}>
              {loading && isBiometric ? (
                <ActivityIndicator color={colors.primary} />
              ) : isBiometric ? (
                <Ionicons name={biometricIcon} size={28} color={colors.primary} />
              ) : (
                <Text
                  style={[
                    styles.numberText,
                    {color: isDelete ? colors.error : colors.text},
                  ]}>
                  {num}
                </Text>
              )}
            </TouchableOpacity>
          );
        })}
      </View>
    );
  };

  const styles = createStyles(colors, isDark);

  return (
    <View style={styles.container}>
      <StatusBar barStyle={isDark ? 'light-content' : 'dark-content'} />
      
      <Animated.View style={[styles.content, {opacity: fadeAnim}]}>
        <View style={[styles.iconContainer, {backgroundColor: colors.card}]}>
          <Ionicons name="shield-checkmark" size={40} color={colors.primary} />
        </View>
        
        <Text style={[styles.title, {color: colors.text}]}>
          Enter Your PIN
        </Text>
        
        <Text style={[styles.subtitle, {color: colors.textSecondary}]}>
          {biometricEnabled 
            ? `Use your PIN or ${Platform.OS === 'android' ? 'fingerprint' : biometricType} to unlock`
            : 'Enter your 6-digit PIN to continue'}
        </Text>

        {renderPinDots()}
        {renderNumberPad()}
      </Animated.View>
    </View>
  );
};

const createStyles = (colors: any, isDark: boolean) => StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
    justifyContent: 'center',
    padding: 24,
  },
  content: {
    alignItems: 'center',
  },
  iconContainer: {
    width: 80,
    height: 80,
    borderRadius: 40,
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 24,
    shadowColor: colors.primary,
    shadowOffset: {width: 0, height: 4},
    shadowOpacity: 0.1,
    shadowRadius: 8,
    elevation: 4,
  },
  title: {
    fontSize: 28,
    fontWeight: '700',
    marginBottom: 8,
    textAlign: 'center',
  },
  subtitle: {
    fontSize: 16,
    textAlign: 'center',
    marginBottom: 32,
    paddingHorizontal: 20,
  },
  dotsContainer: {
    flexDirection: 'row',
    justifyContent: 'center',
    marginBottom: 40,
    gap: 12,
  },
  dot: {
    width: 16,
    height: 16,
    borderRadius: 8,
    borderWidth: 2,
  },
  numberPad: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    justifyContent: 'center',
    maxWidth: 300,
    gap: 16,
  },
  numberButton: {
    width: 70,
    height: 70,
    borderRadius: 35,
    justifyContent: 'center',
    alignItems: 'center',
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 2},
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 2,
  },
  numberText: {
    fontSize: 28,
    fontWeight: '600',
  },
});

export default AppPinEntryScreen;
