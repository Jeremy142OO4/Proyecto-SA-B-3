import { StrictMode } from 'react';
import { ThemeProvider } from '@mui/material/styles';
import { tema } from './styles/tema';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { App } from './app/App';
import { ProveedorAutenticacion } from './features/auth/context/ContextoAutenticacion';
import './styles/global.css';
import './styles/transferencias.css';
import './styles/modal-kyc.css';
import './styles/eventos.css';
import './styles/eventos-scroll.css';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeProvider theme={tema}>
    <BrowserRouter>
      <ProveedorAutenticacion>
        <App />
      </ProveedorAutenticacion>
    </BrowserRouter>
    </ThemeProvider>
  </StrictMode>,
);
