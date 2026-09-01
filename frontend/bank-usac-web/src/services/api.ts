import axios from 'axios';

// La URL base apunta al API Gateway (o al microservicio en desarrollo local)
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Interceptor: inyecta JWT y X-Correlation-ID en cada petición
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('bank_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }

  // Generar correlationId si no viene uno predefinido
  if (!config.headers['X-Correlation-ID']) {
    config.headers['X-Correlation-ID'] = crypto.randomUUID();
  }

  return config;
});

// Interceptor para manejo global de 401 (Sesión expirada)
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response && error.response.status === 401) {
      localStorage.removeItem('bank_token');
      localStorage.removeItem('bank_user');
      if (window.location.pathname !== '/login') {
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);