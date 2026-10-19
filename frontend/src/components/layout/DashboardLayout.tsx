import Box from "@mui/material/Box";
import Toolbar from "@mui/material/Toolbar";
import { useState } from "react";
import { Outlet } from "react-router-dom";

import Navbar from "@/components/layout/Navbar";
import Sidebar, { SIDEBAR_WIDTH } from "@/components/layout/Sidebar";

export default function DashboardLayout() {
  const [mobileOpen, setMobileOpen] = useState(false);

  return (
    <Box sx={{ display: "flex", minHeight: "100vh" }}>
      <Navbar onMenuClick={() => setMobileOpen((prev) => !prev)} />

      <Box component="nav" sx={{ width: { md: SIDEBAR_WIDTH }, flexShrink: { md: 0 } }}>
        <Box sx={{ display: { xs: "none", md: "block" } }}>
          <Sidebar variant="permanent" open onClose={() => undefined} />
        </Box>
        <Box sx={{ display: { xs: "block", md: "none" } }}>
          <Sidebar variant="temporary" open={mobileOpen} onClose={() => setMobileOpen(false)} />
        </Box>
      </Box>

      <Box
        component="main"
        sx={{
          flexGrow: 1,
          width: { md: `calc(100% - ${SIDEBAR_WIDTH}px)` },
          bgcolor: "background.default",
          minHeight: "100vh",
        }}
      >
        <Toolbar />
        <Box sx={{ p: { xs: 2, sm: 3 } }}>
          <Outlet />
        </Box>
      </Box>
    </Box>
  );
}
