import Box from "@mui/material/Box";
import Divider from "@mui/material/Divider";
import Drawer from "@mui/material/Drawer";
import List from "@mui/material/List";
import ListItemButton from "@mui/material/ListItemButton";
import ListItemIcon from "@mui/material/ListItemIcon";
import ListItemText from "@mui/material/ListItemText";
import Toolbar from "@mui/material/Toolbar";
import Typography from "@mui/material/Typography";
import { useLocation, useNavigate } from "react-router-dom";

import { useAppSelector } from "@/app/hooks";
import { NAV_ITEMS } from "@/components/layout/navConfig";

export const SIDEBAR_WIDTH = 260;

interface SidebarProps {
  variant: "permanent" | "temporary";
  open: boolean;
  onClose: () => void;
}

export default function Sidebar({ variant, open, onClose }: SidebarProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const currentRole = useAppSelector((state) => state.auth.user?.role);

  const visibleItems = NAV_ITEMS.filter((item) => !item.roles || !currentRole || item.roles.includes(currentRole));

  const content = (
    <Box role="navigation" aria-label="Main navigation" sx={{ overflow: "auto" }}>
      <Toolbar sx={{ px: 2 }}>
        <Typography variant="h6" noWrap fontWeight={700} color="primary">
          HMS
        </Typography>
      </Toolbar>
      <Divider />
      <List sx={{ px: 1, py: 1 }}>
        {visibleItems.map((item) => {
          const Icon = item.icon;
          const selected = location.pathname.startsWith(item.path);
          return (
            <ListItemButton
              key={item.path}
              selected={selected}
              onClick={() => {
                navigate(item.path);
                if (variant === "temporary") onClose();
              }}
              sx={{ borderRadius: 2, mb: 0.5 }}
            >
              <ListItemIcon sx={{ minWidth: 40 }}>
                <Icon fontSize="small" color={selected ? "primary" : "inherit"} />
              </ListItemIcon>
              <ListItemText
                primary={item.label}
                primaryTypographyProps={{ fontWeight: selected ? 600 : 500 }}
              />
            </ListItemButton>
          );
        })}
      </List>
    </Box>
  );

  return (
    <Drawer
      variant={variant}
      open={variant === "permanent" ? true : open}
      onClose={onClose}
      sx={{
        width: variant === "permanent" ? SIDEBAR_WIDTH : undefined,
        flexShrink: 0,
        [`& .MuiDrawer-paper`]: {
          width: SIDEBAR_WIDTH,
          boxSizing: "border-box",
          borderRight: "1px solid",
          borderColor: "divider",
        },
      }}
    >
      {content}
    </Drawer>
  );
}
