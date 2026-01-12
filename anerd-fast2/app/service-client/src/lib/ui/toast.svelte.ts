export type Toast = {
    id: symbol;
    type: "success" | "error" | "warning" | "info";
    title: string;
    description: string;
    duration: number;
    action?: { label: string; onClick: () => void };
};

class useToast {
    toasts = $state<Toast[]>([]);
    timeoutMap = new Map<symbol, NodeJS.Timeout>();

    showToast(toast: Toast): void {
        this.toasts = [...this.toasts, toast];

        const t = setTimeout(() => {
            this.toasts = this.toasts.filter((t) => t.id !== toast.id);
        }, toast.duration);
        this.timeoutMap.set(toast.id, t);
    }
    removeToast(id: symbol): void {
        this.toasts = this.toasts.filter((t) => t.id !== id);
        const t = this.timeoutMap.get(id);
        if (t) {
            clearTimeout(t);
        }
        this.timeoutMap.delete(id);
    }
    clear(): void {
        for (const t of this.toasts) {
            const timeout = this.timeoutMap.get(t.id);
            if (timeout) {
                clearTimeout(timeout);
            }
        }
        this.toasts = [];
        this.timeoutMap.clear();
    }
    success(title: string, description = ""): void {
        this.showToast({
            id: Symbol(),
            title,
            description,
            type: "success",
            duration: 5000
        });
    }
    error(title: string, description = ""): void {
        this.showToast({
            id: Symbol(),
            title,
            description,
            type: "error",
            duration: 8000
        });
    }
    warning(title: string, description = ""): void {
        this.showToast({
            id: Symbol(),
            title,
            description,
            type: "warning",
            duration: 5000
        });
    }
    info(title: string, description = ""): void {
        this.showToast({
            id: Symbol(),
            title,
            description,
            type: "info",
            duration: 5000
        });
    }
}

export const toast = new useToast();
