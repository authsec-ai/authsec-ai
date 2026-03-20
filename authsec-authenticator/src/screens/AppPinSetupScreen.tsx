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
import {storeAppPin, setPinSetupCompleted, setBiometricEnabled, setAppLockEnabled} from '../services/storage';
import {isBiometricAvailable, getBiometricType} from '../services/biometric';
import {Ionicons} from '@expo/vector-icons';

const AppPinSetupScreen = ({navigation}: any) => {
  const {colors, isDark} = useTheme();
  const [pin, setPin] = useState('');
  const [confirmPin, setConfirmPin] = useState('');
  const [step, setStep] = useState<'create' | 'confirm' | 'biometric'>('create');
  const [loading, setLoading] = useState(false);
  const [biometricAvailable, setBiometricAvailable] = useState(false);
  const [biometricType, setBiometricType] = useState('Biometric');

  const fadeAnim = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    Animated.timing(fadeAnim, {
      toValue: 1,
      duration: 300,
      useNativeDriver: true,
    }).start();

    checkBiometricAvailability();
    
    // Prevent back navigation on Android
    const backHandler = BackHandler.addEventListener('hardwareBackPress', () => {
      return true; // Prevent default back behavior
    });

    return () => backHandler.remove();
  }, [step]);

  const checkBiometricAvailability = async () => {
    const available = await isBiometricAvailable();
    const type = await getBiometricType();
    setBiometricAvailable(available);
    setBiometricType(type);
  };

  const handleNumberPress = (num: string) => {
    if (step === 'create' && pin.length < 6) {
      setPin(pin + num);
    } else if (step === 'confirm' && confirmPin.length < 6) {
      const newPin = confirmPin + num;
      setConfirmPin(newPin);
      
      // Auto-submit when 6 digits entered
      if (newPin.length === 6) {
        verifyPin(newPin);
      }
    }
  };

  const handleDelete = () => {
    if (step === 'create') {
      setPin(pin.slice(0, -1));
    } else if (step === 'confirm') {
      setConfirmPin(confirmPin.slice(0, -1));
    }
  };

  const handleContinue = () => {
    if (pin.length === 6) {
      setStep('confirm');
      fadeAnim.setValue(0);
      Animated.timing(fadeAnim, {
        toValue: 1,
        duration: 300,
        useNativeDriver: true,
      }).start();
    } else {
      Alert.alert('Invalid PIN', 'Please enter a 6-digit PIN');
    }
  };

  const verifyPin = async (enteredPin: string) => {
    if (enteredPin === pin) {
      setLoading(true);
      await storeAppPin(pin);
      await setPinSetupCompleted(true);
      // Enable app lock by default when PIN is set up
      await setAppLockEnabled(true);
      setLoading(false);

      // Check if biometric is available
      const available = await isBiometricAvailable();

      if (available) {
        // Move to biometric setup
        setStep('biometric');
        setConfirmPin('');
        fadeAnim.setValue(0);
        Animated.timing(fadeAnim, {
          toValue: 1,
          duration: 300,
          useNativeDriver: true,
        }).start();
      } else {
        // Skip biometric setup if not available, go directly to home
        navigation.replace('Home');
      }
    } else {
      Alert.alert('PIN Mismatch', 'PINs do not match. Please try again.');
      setConfirmPin('');
      setPin('');
      setStep('create');
      fadeAnim.setValue(0);
      Animated.timing(fadeAnim, {
        toValue: 1,
        duration: 300,
        useNativeDriver: true,
      }).start();
    }
  };

  const handleBiometricSetup = async (enable: boolean) => {
    setLoading(true);
    await setBiometricEnabled(enable);
    setLoading(false);
    
    // Navigate to home
    navigation.replace('Home');
  };

  const renderPinDots = () => {
    const currentPin = step === 'create' ? pin : confirmPin;
    return (
      <View style={styles.dotsContainer}>
        {[0, 1, 2, 3, 4, 5].map((index) => (
          <View
            key={index}
            style={[
              styles.dot,
              {
                backgroundColor: currentPin.length > index ? colors.primary : colors.border,
                borderColor: colors.border,
              },
            ]}
          />
        ))}
      </View>
    );
  };

  const renderNumberPad = () => {
    const numbers = ['1', '2', '3', '4', '5', '6', '7', '8', '9', '', '0', '⌫'];
    
    return (
      <View style={styles.numberPad}>
        {numbers.map((num, index) => {
          if (num === '') {
            return <View key={index} style={styles.numberButton} />;
          }
          
          const isDelete = num === '⌫';
          
          return (
            <TouchableOpacity
              key={index}
              style={[styles.numberButton, {backgroundColor: colors.card}]}
              onPress={() => isDelete ? handleDelete() : handleNumberPress(num)}
              activeOpacity={0.7}>
              <Text style={[styles.numberText, {color: isDelete ? colors.error : colors.text}]}>
                {num}
              </Text>
            </TouchableOpacity>
          );
        })}
      </View>
    );
  };

  const renderBiometricSetup = () => {
    const isAndroid = Platform.OS === 'android';

    let titleLabel = biometricType;
    let descriptionLabel = `Use ${biometricType} for quick and secure access to the app`;
    let iconName: keyof typeof Ionicons.glyphMap = 'finger-print';

    if (isAndroid) {
      titleLabel = 'Fingerprint';
      descriptionLabel = 'Use your fingerprint for quick and secure access to the app';
      iconName = 'finger-print';
    } else {
      if (biometricType === 'Face ID') {
        titleLabel = 'Face ID';
        descriptionLabel = 'Use Face ID for quick and secure access to the app';
        iconName = 'person-circle-outline';
      } else {
        // Default to Touch ID style for other iOS biometrics
        titleLabel = 'Touch ID';
        descriptionLabel = 'Use Touch ID for quick and secure access to the app';
        iconName = 'finger-print';
      }
    }

    return (
      <Animated.View style={[styles.biometricContainer, {opacity: fadeAnim}]}>
        <View style={[styles.iconContainer, {backgroundColor: colors.card}]}>
          <Ionicons name={iconName} size={40} color={colors.primary} />
        </View>
        
        <Text style={[styles.title, {color: colors.text}]}>
          Enable {titleLabel}?
        </Text>
        
        <Text style={[styles.subtitle, {color: colors.textSecondary}]}>
          {descriptionLabel}
        </Text>

        <TouchableOpacity
          style={[styles.biometricButton, {backgroundColor: colors.primary}]}
          onPress={() => handleBiometricSetup(true)}
          disabled={loading}>
          {loading ? (
            <ActivityIndicator color={isDark ? '#000' : '#fff'} />
          ) : (
            <Text style={[styles.biometricButtonText, {color: isDark ? '#000' : '#fff'}]}>
              Enable {titleLabel}
            </Text>
          )}
        </TouchableOpacity>

        <TouchableOpacity
          style={styles.skipButton}
          onPress={() => handleBiometricSetup(false)}
          disabled={loading}>
          <Text style={[styles.skipButtonText, {color: colors.textSecondary}]}>
            Skip for now
          </Text>
        </TouchableOpacity>
      </Animated.View>
    );
  };

  const styles = createStyles(colors, isDark);

  if (step === 'biometric' && biometricAvailable) {
    return (
      <View style={styles.container}>
        <StatusBar barStyle={isDark ? 'light-content' : 'dark-content'} />
        {renderBiometricSetup()}
      </View>
    );
  }

  if (step === 'biometric' && !biometricAvailable) {
    // If biometric not available, go straight to home
    setTimeout(() => navigation.replace('Home'), 0);
    return null;
  }

  return (
    <View style={styles.container}>
      <StatusBar barStyle={isDark ? 'light-content' : 'dark-content'} />
      
      <Animated.View style={[styles.content, {opacity: fadeAnim}]}>
        <View style={[styles.iconContainer, {backgroundColor: colors.card}]}>
          <Ionicons name="lock-closed" size={40} color={colors.primary} />
        </View>
        
        <Text style={[styles.title, {color: colors.text}]}>
          {step === 'create' ? 'Create App PIN' : 'Confirm Your PIN'}
        </Text>
        
        <Text style={[styles.subtitle, {color: colors.textSecondary}]}>
          {step === 'create' 
            ? 'Set a 6-digit PIN to secure your app'
            : 'Re-enter your PIN to confirm'}
        </Text>

        {renderPinDots()}
        {renderNumberPad()}

        {step === 'create' && pin.length === 6 && (
          <TouchableOpacity
            style={[styles.continueButton, {backgroundColor: colors.primary}]}
            onPress={handleContinue}>
            <Text style={[styles.continueButtonText, {color: isDark ? '#000' : '#fff'}]}>
              Continue
            </Text>
          </TouchableOpacity>
        )}
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
    ...(Platform.OS === 'ios' ? {
      shadowColor: colors.primary,
      shadowOffset: {width: 0, height: 4},
      shadowOpacity: 0.1,
      shadowRadius: 8,
    } : {}),
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
  continueButton: {
    marginTop: 32,
    paddingHorizontal: 48,
    paddingVertical: 16,
    borderRadius: 12,
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 4},
    shadowOpacity: 0.2,
    shadowRadius: 8,
    elevation: 4,
  },
  continueButtonText: {
    fontSize: 18,
    fontWeight: '700',
  },
  biometricContainer: {
    alignItems: 'center',
  },
  biometricButton: {
    width: '100%',
    paddingVertical: 18,
    borderRadius: 12,
    alignItems: 'center',
    marginTop: 32,
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 4},
    shadowOpacity: 0.2,
    shadowRadius: 8,
    elevation: 4,
  },
  biometricButtonText: {
    fontSize: 18,
    fontWeight: '700',
  },
  skipButton: {
    marginTop: 16,
    padding: 12,
  },
  skipButtonText: {
    fontSize: 16,
    fontWeight: '600',
  },
});

export default AppPinSetupScreen;
