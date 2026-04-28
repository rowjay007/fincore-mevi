"use client"

import * as React from "react"
import { Shield, Fingerprint, Trash2, Key, Loader2, Plus, AlertCircle } from "lucide-react"
import { startRegistration, startAuthentication } from "@simplewebauthn/browser"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Badge } from "@/components/ui/badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Skeleton } from "@/components/ui/skeleton"

interface CredentialInfo {
  id: string
  created_at: string
  updated_at: string
  sign_count: number
}

export function PasskeyManager() {
  const [credentials, setCredentials] = React.useState<CredentialInfo[]>([])
  const [loading, setLoading] = React.useState(true)
  const [registering, setRegistering] = React.useState(false)
  const [deletingId, setDeletingId] = React.useState<string | null>(null)

  const fetchCredentials = React.useCallback(async () => {
    try {
      const res = await fetch("/webauthn/credentials")
      if (res.ok) {
        const data = await res.json()
        setCredentials(data || [])
      }
    } catch (err) {
      console.error("Failed to fetch credentials", err)
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    fetchCredentials()
  }, [fetchCredentials])

  const handleRegister = async () => {
    setRegistering(true)
    try {
      // 1. Get options from server
      const resp = await fetch("/webauthn/register/begin")
      if (!resp.ok) {
        const text = await resp.text()
        throw new Error(text || "Failed to begin registration")
      }
      const options = await resp.json()

      // 2. Start WebAuthn registration
      const attestation = await startRegistration(options)

      // 3. Finish registration on server
      const finishResp = await fetch("/webauthn/register/finish", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(attestation),
      })

      if (!finishResp.ok) throw new Error("Failed to finish registration")

      toast.success("Passkey registered", {
        description: "You can now use this passkey for secure login.",
      })
      fetchCredentials()
    } catch (err: any) {
      console.error(err)
      toast.error("Registration failed", {
        description: err.message || "An error occurred during passkey registration.",
      })
    } finally {
      setRegistering(false)
    }
  }

  const handleDelete = async (id: string) => {
    setDeletingId(id)
    try {
      const res = await fetch(`/webauthn/credentials/delete?id=${id}`, {
        method: "DELETE",
      })
      if (!res.ok) throw new Error("Failed to delete passkey")

      toast.success("Passkey removed")
      setCredentials((prev) => prev.filter((c) => c.id !== id))
    } catch (err: any) {
      toast.error("Deletion failed", {
        description: err.message,
      })
    } finally {
      setDeletingId(null)
    }
  }

  return (
    <Card className="border-border/50 shadow-xl overflow-hidden">
      <CardHeader className="bg-muted/30 border-b">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Fingerprint className="h-5 w-5 text-primary" />
            <div>
              <CardTitle className="text-lg">Passkey Management</CardTitle>
              <CardDescription>Secure passwordless authentication</CardDescription>
            </div>
          </div>
          <Button
            size="sm"
            onClick={handleRegister}
            disabled={registering}
            className="gap-2"
          >
            {registering ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Plus className="h-4 w-4" />
            )}
            Add Passkey
          </Button>
        </div>
      </CardHeader>
      <CardContent className="p-0">
        {loading ? (
          <div className="p-6 space-y-4">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : credentials.length === 0 ? (
          <div className="flex flex-col items-center justify-center p-12 text-center text-muted-foreground">
            <div className="bg-muted p-4 rounded-full mb-4">
              <Key className="h-8 w-8 opacity-20" />
            </div>
            <p className="text-sm font-medium">No passkeys registered</p>
            <p className="text-xs max-w-[200px] mt-1">
              Add a passkey to enable secure, phishing-resistant login.
            </p>
          </div>
        ) : (
          <Table>
            <TableHeader className="bg-muted/10">
              <TableRow className="border-none">
                <TableHead className="uppercase text-[10px] font-bold tracking-tighter pl-6">ID</TableHead>
                <TableHead className="uppercase text-[10px] font-bold tracking-tighter">Created</TableHead>
                <TableHead className="uppercase text-[10px] font-bold tracking-tighter">Usage</TableHead>
                <TableHead className="text-right pr-6"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {credentials.map((cred) => (
                <TableRow key={cred.id} className="group border-b border-border/30 last:border-0 hover:bg-muted/20">
                  <TableCell className="font-mono text-[10px] font-bold pl-6 text-primary">
                    {cred.id.slice(0, 12)}...
                  </TableCell>
                  <TableCell className="text-xs font-medium text-muted-foreground">
                    {new Date(cred.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline" className="text-[9px] font-mono px-1.5 py-0">
                      {cred.sign_count} USES
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right pr-6">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 text-destructive opacity-0 group-hover:opacity-100 transition-opacity"
                      onClick={() => handleDelete(cred.id)}
                      disabled={deletingId === cred.id}
                    >
                      {deletingId === cred.id ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <Trash2 className="h-4 w-4" />
                      )}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
      <CardFooter className="bg-muted/10 border-t py-3 px-6">
        <div className="flex items-center gap-2 text-[10px] font-medium text-muted-foreground">
          <Shield className="h-3 w-3 text-emerald-500" />
          FIDO2 / WebAuthn Certified Compliance
        </div>
      </CardFooter>
    </Card>
  )
}
