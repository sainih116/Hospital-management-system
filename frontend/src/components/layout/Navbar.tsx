import DarkModeRoundedIcon from "@mui/icons-material/DarkModeRounded";
import LightModeRoundedIcon from "@mui/icons-material/LightModeRounded";
import LogoutRoundedIcon from "@mui/icons-material/LogoutRounded";
import MenuRoundedIcon from "@mui/icons-material/MenuRounded";
import AppBar from "@mui/material/AppBar";
import Avatar from "@mui/material/Avatar";
import Box from "@mui/material/Box";
import IconButton from "@mui/material/IconButton";
import ListItemIcon from "@mui/material/ListItemIcon";
import Menu from "@mui/material/Menu";
import MenuItem from "@mui/material/MenuItem";
import Stack from "@mui/material/Stack";
import Toolbar from "@mui/material/Toolbar";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useState, type MouseEvent } from "react";

import { useAppDispatch, useAppSelector } from "@/app/hooks";
import { logout } from "@/features/auth/authSlice";
import { toggleSidebar, toggleThemeMode } from "@/features/ui/uiSlice";
import { SIDEBAR_WIDTH } from "@/components/layout/Sidebar";

interface NavbarProps {
  onMenuClick: () => void;
}

export default function Navbar({ onMenuClick }: NavbarProps) {
  const dispatch = useAppDispatch();
  const themeMode = useAppSelector((state) => state.ui.themeMode);
  const user = useAppSelector((state) => state.auth.user);

  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
  const menuOpen = Boolean(anchorEl);

  const handleAvatarClick = (event: MouseEvent<HTMLElement>) => setAnchorEl(event.currentTarget);
  const handleMenuClose = () => setAnchorEl(null);
  const handleLogout = () => {
    handleMenuClose();
    void dispatch(logout());
  };

  return (
    <AppBar
      position="fixed"
      color="inherit"
      elevation={0}
      sx={{
        width: { md: `calc(100% - ${SIDEBAR_WIDTH}px)` },
        ml: { md: `${SIDEBAR_WIDTH}px` },
        borderBottom: "1px solid",
        borderColor: "divider",
        backgroundColor: "background.paper",
      }}
    >
      <Toolbar sx={{ gap: 1 }}>
        <IconButton
          color="inherit"
          aria-label="Toggle navigation menu"
          onClick={() => {
            onMenuClick();
            dispatch(toggleSidebar());
          }}
          sx={{ display: { md: "none" } }}
        >
          <MenuRoundedIcon />
        </IconButton>

        <Typography variant="subtitle1" fontWeight={600} sx={{ flexGrow: 1 }}>
          Hospital Management System
        </Typography>

        <Stack direction="row" spacing={1} alignItems="center">
          <Tooltip title={themeMode === "dark" ? "Switch to light mode" : "Switch to dark mode"}>
            <IconButton color="inherit" onClick={() => dispatch(toggleThemeMode())}>
              {themeMode === "dark" ? <LightModeRoundedIcon /> : <DarkModeRoundedIcon />}
            </IconButton>
          </Tooltip>

          <Tooltip title={user ? user.name : "Account"}>
            <IconButton onClick={handleAvatarClick} size="small" sx={{ ml: 1 }}>
              <Avatar sx={{ width: 34, height: 34, bgcolor: "primary.main" }}>
                {user ? user.name.charAt(0).toUpperCase() : "?"}
              </Avatar>
            </IconButton>
          </Tooltip>

          <Menu anchorEl={anchorEl} open={menuOpen} onClose={handleMenuClose} onClick={handleMenuClose}>
            <Box sx={{ px: 2, py: 1 }}>
              <Typography variant="body2" fontWeight={600}>
                {user?.name ?? "Guest"}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {user?.role ?? "Not signed in"}
              </Typography>
            </Box>
            <MenuItem onClick={handleLogout}>
              <ListItemIcon>
                <LogoutRoundedIcon fontSize="small" />
              </ListItemIcon>
              Logout
            </MenuItem>
          </Menu>
        </Stack>
      </Toolbar>
    </AppBar>
  );
}
