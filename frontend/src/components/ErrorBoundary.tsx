import React from "react";
import { useTranslation } from "react-i18next";

interface ErrorBoundaryProps {
  children: React.ReactNode;
}

interface ErrorBoundaryState {
  error: Error | null;
}

const ErrorFallback: React.FC = () => {
  const { t } = useTranslation();
  return (
    <div className="flex min-h-screen w-full flex-col items-center justify-center gap-4 bg-background px-6 text-center text-foreground">
      <h1 className="text-xl font-semibold">{t("app.error_boundary.title")}</h1>
      <p className="max-w-md text-sm text-default-500">
        {t("app.error_boundary.description")}
      </p>
      <button
        type="button"
        onClick={() => window.location.reload()}
        className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground"
      >
        {t("app.error_boundary.reload")}
      </button>
    </div>
  );
};

// Top-level render guard: a throw anywhere in the app tree would
// otherwise unmount the whole UI to a blank window. The fallback offers
// a reload, which re-runs the normal startup path.
export class ErrorBoundary extends React.Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  state: ErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error("[ErrorBoundary] render error", error, info);
  }

  render() {
    if (this.state.error) {
      return <ErrorFallback />;
    }
    return this.props.children;
  }
}

export default ErrorBoundary;
