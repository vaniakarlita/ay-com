export const SECURITY_QUESTIONS = [
  'What was the name of your first pet?',
  'What city were you born in?',
  'What is your favorite video game?',
  'What was the name of your first school?',
  'What was your childhood nickname?',
];

export const MAX_IMAGE_SIZE = 2 * 1024 * 1024;

export function validateName(value: string): string {
  const v = value.trim();
  if (!v) return 'Name is required';
  if (v.length <= 4) return 'Name must be more than 4 characters';
  if (!/^[\p{L}\s]+$/u.test(v)) return 'Name must not contain symbols or numbers';
  return '';
}

export function validateUsername(value: string): string {
  if (!value) return 'Username is required';
  if (!/^[a-zA-Z0-9_]{3,15}$/.test(value)) {
    return 'Username must be 3-15 characters: letters, numbers, or underscore';
  }
  return '';
}

export function validateEmail(value: string): string {
  if (!value) return 'Email is required';
  if (!/^[^\s@]+@[^\s@.]+\.com$/.test(value)) {
    return 'Email must match [name]@[domain].com';
  }
  return '';
}

export const PASSWORD_RULES = [
  { label: 'At least 8 characters', test: (v: string) => v.length >= 8 },
  { label: 'One uppercase letter', test: (v: string) => /[A-Z]/.test(v) },
  { label: 'One lowercase letter', test: (v: string) => /[a-z]/.test(v) },
  { label: 'One number', test: (v: string) => /\d/.test(v) },
  { label: 'One symbol (e.g. !@#$%)', test: (v: string) => /[^A-Za-z0-9]/.test(v) },
];

export function validatePassword(value: string): string {
  if (!value) return 'Password is required';
  const failed = PASSWORD_RULES.find((r) => !r.test(value));
  return failed ? `Password needs: ${failed.label.toLowerCase()}` : '';
}

export function validateConfirm(password: string, confirm: string): string {
  if (!confirm) return 'Please confirm your password';
  if (password !== confirm) return 'Passwords do not match';
  return '';
}

export function validateGender(value: string): string {
  return value === 'male' || value === 'female' ? '' : 'Please select your gender';
}

export function calculateAge(dob: string): number {
  const birth = new Date(dob);
  const today = new Date();
  let age = today.getFullYear() - birth.getFullYear();
  const hadBirthday =
    today.getMonth() > birth.getMonth() ||
    (today.getMonth() === birth.getMonth() && today.getDate() >= birth.getDate());
  if (!hadBirthday) age--;
  return age;
}

export function validateDob(value: string): string {
  if (!value) return 'Date of birth is required';
  if (Number.isNaN(new Date(value).getTime())) return 'Invalid date';
  if (calculateAge(value) < 13) return 'You must be at least 13 years old';
  return '';
}

export function validateImage(file: File | null, label: string): string {
  if (!file) return `${label} is required`;
  if (!file.type.startsWith('image/')) return `${label} must be an image`;
  if (file.size > MAX_IMAGE_SIZE) return `${label} must be 2 MB or smaller`;
  return '';
}

export function validateQuestion(question: string, answer: string): string {
  if (!SECURITY_QUESTIONS.includes(question)) return 'Please choose a security question';
  if (!answer.trim()) return 'Please answer your security question';
  return '';
}