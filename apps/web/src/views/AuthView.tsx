import React, { useState } from "react";
import { ApiError } from "../api/client";
import { AlertToast } from "../components/ui/AlertToast";
import { Button } from "../components/ui/Button";
import { Input } from "../components/ui/Input";
import { useAuth } from "../context/AuthContext";

export function AuthView() {
  const { login, register } = useAuth();
  const [tab, setTab] = useState<"login" | "register">("login");

  // Login form state
  const [loginEmail, setLoginEmail] = useState("");
  const [loginPassword, setLoginPassword] = useState("");

  // Register form state
  const [regName, setRegName] = useState("");
  const [regEmail, setRegEmail] = useState("");
  const [regPassword, setRegPassword] = useState("");
  const [regTerms, setRegTerms] = useState(false);

  const [isLoading, setIsLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const handleLoginSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMessage(null);
    setFieldErrors({});

    if (!loginEmail.trim()) {
      setFieldErrors({ email: "Email wajib diisi" });
      return;
    }
    if (!loginPassword) {
      setFieldErrors({ password: "Kata sandi wajib diisi" });
      return;
    }

    setIsLoading(true);
    try {
      await login(loginEmail, loginPassword);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setErrorMessage(err.message || "Gagal masuk. Periksa email dan kata sandi.");
        if (err.fields) {
          setFieldErrors(err.fields);
        }
      } else {
        setErrorMessage("Terjadi kesalahan jaringan.");
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleRegisterSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMessage(null);
    setFieldErrors({});

    const errors: Record<string, string> = {};
    if (!regName.trim()) errors.name = "Nama tampilan wajib diisi";
    if (!regEmail.trim()) errors.email = "Email wajib diisi";
    if (regPassword.length < 8) {
      errors.password = "Kata sandi minimal 8 karakter";
    }
    if (!regTerms) {
      errors.terms = "Anda harus menyetujui ketentuan layanan";
    }

    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      return;
    }

    setIsLoading(true);
    try {
      await register(regEmail, regPassword, regName, regTerms);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setErrorMessage(err.message || "Pendaftaran gagal.");
        if (err.fields) {
          setFieldErrors(err.fields);
        }
      } else {
        setErrorMessage("Terjadi kesalahan jaringan.");
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen flex-col justify-center py-12 px-4 sm:px-6 lg:px-8 bg-slate-50">
      <div className="sm:mx-auto sm:w-full sm:max-w-md">
        <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-slate-900 text-white font-bold text-xl shadow-md">
          B
        </div>
        <h2 className="mt-4 text-center text-2xl font-bold tracking-tight text-slate-900">
          BorrowHub
        </h2>
        <p className="mt-1 text-center text-sm text-slate-500">
          Manajemen peminjaman peralatan organisasi yang terstruktur
        </p>
      </div>

      <div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
        <div className="bg-white py-8 px-6 shadow-sm ring-1 ring-slate-900/5 sm:rounded-2xl sm:px-10">
          {/* Switch Tab */}
          <div className="flex rounded-lg bg-slate-100 p-1 mb-6">
            <button
              type="button"
              onClick={() => {
                setTab("login");
                setErrorMessage(null);
                setFieldErrors({});
              }}
              className={`w-1/2 py-2 text-sm font-semibold rounded-md transition-colors cursor-pointer ${
                tab === "login"
                  ? "bg-white text-slate-900 shadow-xs"
                  : "text-slate-600 hover:text-slate-900"
              }`}
            >
              Masuk
            </button>
            <button
              type="button"
              onClick={() => {
                setTab("register");
                setErrorMessage(null);
                setFieldErrors({});
              }}
              className={`w-1/2 py-2 text-sm font-semibold rounded-md transition-colors cursor-pointer ${
                tab === "register"
                  ? "bg-white text-slate-900 shadow-xs"
                  : "text-slate-600 hover:text-slate-900"
              }`}
            >
              Daftar
            </button>
          </div>

          {errorMessage && (
            <div className="mb-5">
              <AlertToast
                type="error"
                message={errorMessage}
                onClose={() => setErrorMessage(null)}
              />
            </div>
          )}

          {tab === "login" ? (
            <form onSubmit={handleLoginSubmit} className="space-y-4">
              <Input
                label="Email"
                type="email"
                placeholder="nama@organisasi.id"
                value={loginEmail}
                onChange={(e) => setLoginEmail(e.target.value)}
                error={fieldErrors.email}
                autoComplete="email"
                required
              />
              <Input
                label="Kata Sandi"
                type="password"
                placeholder="••••••••"
                value={loginPassword}
                onChange={(e) => setLoginPassword(e.target.value)}
                error={fieldErrors.password}
                autoComplete="current-password"
                required
              />
              <Button
                type="submit"
                variant="primary"
                className="w-full mt-2"
                isLoading={isLoading}
              >
                Masuk ke Akun
              </Button>
            </form>
          ) : (
            <form onSubmit={handleRegisterSubmit} className="space-y-4">
              <Input
                label="Nama Lengkap"
                type="text"
                placeholder="Budi Santoso"
                value={regName}
                onChange={(e) => setRegName(e.target.value)}
                error={fieldErrors.name || fieldErrors.display_name}
                autoComplete="name"
                required
              />
              <Input
                label="Email"
                type="email"
                placeholder="budi@organisasi.id"
                value={regEmail}
                onChange={(e) => setRegEmail(e.target.value)}
                error={fieldErrors.email}
                autoComplete="email"
                required
              />
              <Input
                label="Kata Sandi"
                type="password"
                placeholder="Minimal 8 karakter"
                value={regPassword}
                onChange={(e) => setRegPassword(e.target.value)}
                error={fieldErrors.password}
                autoComplete="new-password"
                required
              />
              <div>
                <label className="flex items-start gap-2 text-xs text-slate-600 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={regTerms}
                    onChange={(e) => setRegTerms(e.target.checked)}
                    className="mt-0.5 rounded border-slate-300 text-slate-900 focus:ring-slate-900"
                  />
                  <span>
                    Saya menyetujui ketentuan peminjaman dan kebijakan
                    penggunaan peralatan BorrowHub.
                  </span>
                </label>
                {fieldErrors.terms && (
                  <p className="mt-1 text-xs text-rose-600">
                    {fieldErrors.terms}
                  </p>
                )}
              </div>
              <Button
                type="submit"
                variant="primary"
                className="w-full mt-2"
                isLoading={isLoading}
              >
                Buat Akun Baru
              </Button>
            </form>
          )}
        </div>
      </div>
    </div>
  );
}
