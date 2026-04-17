import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { Toaster } from "@/components/ui/sonner"
import { SidebarProvider, SidebarInset, SidebarTrigger } from "@/components/ui/sidebar"
import { AppSidebar } from "@/components/app-sidebar"
import { TooltipProvider } from "@/components/ui/tooltip"
import { Badge } from "@/components/ui/badge"
import Providers from "@/components/providers"
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "FinCore Master Dashboard",
  description: "Enterprise Banking Ecosystem Monitoring",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="dark" suppressHydrationWarning>
      <body className={`${geistSans.variable} ${geistMono.variable} bg-background text-foreground antialiased min-h-screen`} suppressHydrationWarning>
        <Providers>
          <TooltipProvider>
            <SidebarProvider>
              <AppSidebar />
              <SidebarInset>
                <header className="flex h-16 shrink-0 items-center gap-2 border-b px-4 backdrop-blur-md bg-background/80 sticky top-0 z-50">
                  <SidebarTrigger className="-ml-1" />
                  <div className="flex items-center gap-4 px-4 w-full justify-between">
                    <span className="text-sm font-bold tracking-tight text-blue-500 font-mono">FINCORE_OS v1.2.0</span>
                    <div className="hidden md:flex items-center gap-4">
                      <div className="flex items-center gap-2">
                        <div className="h-2 w-2 bg-emerald-500 rounded-full animate-pulse shadow-[0_0_8px_rgba(16,185,129,0.8)]" />
                        <span className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground">Gateway Active</span>
                      </div>
                      <Badge variant="outline" className="text-[10px] font-mono border-blue-500/30 text-blue-400">PROD-01-EUS</Badge>
                    </div>
                  </div>
                </header>
                <main className="flex-1 overflow-auto p-4 md:p-8">
                  {children}
                </main>
              </SidebarInset>
            </SidebarProvider>
          </TooltipProvider>
        </Providers>
        <Toaster position="top-right" expand={true} richColors />
      </body>
    </html>
  );
}
