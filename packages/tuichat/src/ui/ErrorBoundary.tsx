import { Component, type ReactNode } from "react";

interface ErrorBoundaryProps {
  onError: (error: unknown) => void;
  children: ReactNode;
}

interface ErrorBoundaryState {
  errored: boolean;
}

export class ErrorBoundary extends Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  state: ErrorBoundaryState = { errored: false };

  static getDerivedStateFromError(): ErrorBoundaryState {
    return { errored: true };
  }

  componentDidCatch(error: unknown): void {
    this.props.onError(error);
  }

  render(): ReactNode {
    if (this.state.errored) return null;
    return this.props.children;
  }
}
