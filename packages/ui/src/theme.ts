import { createTheme } from '@mui/material/styles';
import { zhCN } from '@mui/material/locale';

// CubeBox 2deaf927 experience baseline: teal header, accessible dark buttons,
// rounded surfaces. This is a new UI-only implementation, not a domain port.
export const theme = createTheme({
  palette: {
    primary: { main: '#066b68', contrastText: '#ffffff' },
    info: { main: '#0069a4' },
    background: { default: '#f8fafb', paper: '#ffffff' },
  },
  shape: { borderRadius: 10 },
  typography: { fontFamily: 'Inter, "CHNHCM CJK", system-ui, -apple-system, "Segoe UI", sans-serif' },
  components: {
    MuiButton: {
      styleOverrides: {
        root: { textTransform: 'none' },
        containedPrimary: { backgroundColor: '#087f7b', '&:hover': { backgroundColor: '#066b68' } },
        textPrimary: { color: '#087f7b' },
        outlinedPrimary: { color: '#087f7b', borderColor: '#087f7b' },
      },
    },
    MuiButtonBase: { styleOverrides: { root: { '&.Mui-focusVisible': { outline: '2px solid currentColor', outlineOffset: 2 } } } },
    MuiTab: { styleOverrides: { root: { textTransform: 'none' } } },
    MuiChip: { styleOverrides: { root: { '&.MuiChip-colorPrimary.MuiChip-outlined': { color: '#087f7b', borderColor: '#087f7b' } } } },
    MuiAppBar: { styleOverrides: { root: { boxShadow: 'none', borderBottom: '1px solid #e5e7eb' } } },
  },
}, zhCN);
