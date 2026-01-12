<script lang="ts">
    const {
        id,
        title,
        message,
        confirmText = "Delete"
    } = $props<{
        id: string;
        title: string;
        message: string;
        confirmText?: string;
    }>();
</script>

<el-dialog>
    <dialog
        {id}
        aria-labelledby="dialog-title"
        class="fixed inset-0 size-auto max-h-none max-w-none overflow-y-auto bg-transparent backdrop:bg-transparent"
    >
        <el-dialog-backdrop
            class="fixed inset-0 bg-black/50 transition-opacity data-closed:opacity-0 data-enter:duration-300 data-enter:ease-out data-leave:duration-200 data-leave:ease-in"
        ></el-dialog-backdrop>

        <div
            class="flex min-h-full items-end justify-center p-4 text-center focus:outline-none sm:items-center sm:p-0"
        >
            <el-dialog-panel
                class="relative transform overflow-hidden rounded-lg bg-base-100 px-4 pt-5 pb-4 text-left shadow-xl transition-all data-closed:translate-y-4 data-closed:opacity-0 data-enter:duration-300 data-enter:ease-out data-leave:duration-200 data-leave:ease-in sm:my-8 sm:w-full sm:max-w-lg sm:p-6 data-closed:sm:translate-y-0 data-closed:sm:scale-95"
            >
                <div class="sm:flex sm:items-start">
                    <div
                        class="mx-auto flex size-12 shrink-0 items-center justify-center rounded-full bg-error sm:mx-0 sm:size-10"
                    >
                        <svg
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="1.5"
                            data-slot="icon"
                            aria-hidden="true"
                            class="size-6 text-error-content"
                        >
                            <path
                                d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                            />
                        </svg>
                    </div>
                    <div class="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left">
                        <h3 id="dialog-title" class="text-base font-semibold text-base-content">
                            {title}
                        </h3>
                        <div class="mt-2">
                            <p class="text-sm text-base-content/70">
                                {message}
                            </p>
                        </div>
                    </div>
                </div>
                <div class="mt-5 sm:mt-4 sm:flex sm:flex-row-reverse">
                    <button type="submit" class="btn btn-error sm:ml-3">{confirmText}</button>
                    <button
                        type="button"
                        command="close"
                        commandfor={id}
                        class="btn btn-ghost mt-3 sm:mt-0">Cancel</button
                    >
                </div>
            </el-dialog-panel>
        </div>
    </dialog>
</el-dialog>
