import { createTheme } from '@mui/material/styles';

export const tema = createTheme({
  palette: {
    primary: { main: '#367478', dark: '#28575b', contrastText: '#ffffff' },
    secondary: { main: '#d58e60', contrastText: '#292c33' },
    background: { default: '#f4f0e7', paper: '#ffffff' },
    text: { primary: '#292c33', secondary: '#625346' },
    divider: '#d7d6c8',
    warning: { main: '#96603d', light: '#f5e6cc', dark: '#6a4935' },
    error: { main: '#973c35', light: '#f7e3de' },
    success: { main: '#285f52', light: '#e0efea' },
    info: { main: '#367478', light: '#e1f0f2' },
  },
  typography: { fontFamily: 'Inter, ui-sans-serif, system-ui, sans-serif' },
});
