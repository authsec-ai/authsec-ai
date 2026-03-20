export interface PasswordValidation {
  isValid: boolean;
  errors: string[];
  requirements: PasswordRequirement[];
}

export interface PasswordRequirement {
  text: string;
  met: boolean;
}

export const validatePassword = (password: string): PasswordValidation => {
  const requirements: PasswordRequirement[] = [
    {
      text: "At least 10 characters",
      met: password.length >= 10,
    },
    {
      text: "Contains uppercase letter",
      met: /[A-Z]/.test(password),
    },
    {
      text: "Contains lowercase letter",
      met: /[a-z]/.test(password),
    },
    {
      text: "Contains number",
      met: /\d/.test(password),
    },
    {
      text: "Contains special character (!@#$%^&*...)",
      met: /[!@#$%^&*(),.?":{}|<>]/.test(password),
    },
  ];

  const errors = requirements
    .filter(req => !req.met)
    .map(req => req.text);

  return {
    isValid: requirements.every(req => req.met),
    errors,
    requirements,
  };
};

export const getPasswordStrength = (password: string): {
  strength: 'weak' | 'fair' | 'good' | 'strong';
  score: number;
} => {
  let metCount = 0;
  const totalCriteria = 5;

  if (password.length >= 10) metCount++;
  if (/[A-Z]/.test(password)) metCount++;
  if (/[a-z]/.test(password)) metCount++;
  if (/\d/.test(password)) metCount++;
  if (/[!@#$%^&*(),.?":{}|<>]/.test(password)) metCount++;

  const score = (metCount / totalCriteria) * 100;

  let strength: 'weak' | 'fair' | 'good' | 'strong';
  if (score < 40) strength = 'weak';
  else if (score < 60) strength = 'fair';
  else if (score < 80) strength = 'good';
  else strength = 'strong';

  return { strength, score };
};