const foundations = [
  "Bun workspace",
  "React 19",
  "TypeScript",
  "Vite",
  "Tailwind CSS",
];

export function App() {
  return (
    <main className="min-h-screen px-6 py-16 sm:px-10">
      <section className="mx-auto max-w-4xl">
        <p className="mb-4 text-sm font-semibold uppercase tracking-widest text-slate-500">
          BorrowHub
        </p>

        <h1 className="max-w-3xl text-4xl font-semibold tracking-tight text-slate-950 sm:text-6xl">
          Peminjaman peralatan yang jelas dan dapat dipercaya.
        </h1>

        <p className="mt-6 max-w-2xl text-lg leading-8 text-slate-600">
          Foundation frontend berhasil dijalankan. Fitur produk akan ditambahkan
          secara bertahap berdasarkan contract dan domain BorrowHub.
        </p>

        <div className="mt-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {foundations.map((foundation) => (
            <div
              key={foundation}
              className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm"
            >
              <p className="font-medium text-slate-900">{foundation}</p>

              <p className="mt-1 text-sm text-slate-500">
                Terpasang dan siap digunakan.
              </p>
            </div>
          ))}
        </div>
      </section>
    </main>
  );
}
