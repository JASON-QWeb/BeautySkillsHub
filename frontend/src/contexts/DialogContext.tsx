import React, { createContext, useContext, useState, useCallback, ReactNode } from 'react'

type DialogType = 'alert' | 'confirm'

interface DialogState {
    isOpen: boolean
    type: DialogType
    title: string
    message: string
    resolve: ((value: boolean) => void) | null
}

interface DialogContextValue {
    showAlert: (message: string, title?: string) => Promise<void>
    showConfirm: (message: string, title?: string) => Promise<boolean>
}

const DialogContext = createContext<DialogContextValue | undefined>(undefined)

export function DialogProvider({ children }: { children: ReactNode }) {
    const [dialog, setDialog] = useState<DialogState>({
        isOpen: false,
        type: 'alert',
        title: '',
        message: '',
        resolve: null,
    })

    const showAlert = useCallback((message: string, title: string = '提示') => {
        return new Promise<void>((resolve) => {
            setDialog({
                isOpen: true,
                type: 'alert',
                title,
                message,
                resolve: () => resolve(),
            })
        })
    }, [])

    const showConfirm = useCallback((message: string, title: string = '确认') => {
        return new Promise<boolean>((resolve) => {
            setDialog({
                isOpen: true,
                type: 'confirm',
                title,
                message,
                resolve,
            })
        })
    }, [])

    const handleClose = (value: boolean) => {
        setDialog(prev => ({ ...prev, isOpen: false }))
        if (dialog.resolve) {
            dialog.resolve(value)
        }
    }

    return (
        <DialogContext.Provider value={{ showAlert, showConfirm }}>
            {children}
            {dialog.isOpen && (
                <div className="dialog-overlay" onClick={() => handleClose(false)}>
                    <div className="dialog-modal glass-card" onClick={e => e.stopPropagation()}>
                        <div className="dialog-header">
                            <h3 className="dialog-title">{dialog.title}</h3>
                            <button className="dialog-close" onClick={() => handleClose(false)}>×</button>
                        </div>
                        <div className="dialog-body">
                            <p>{dialog.message}</p>
                        </div>
                        <div className="dialog-footer">
                            {dialog.type === 'confirm' && (
                                <button className="btn btn-secondary" onClick={() => handleClose(false)}>
                                    取消
                                </button>
                            )}
                            <button className="btn btn-primary" onClick={() => handleClose(true)}>
                                确定
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </DialogContext.Provider>
    )
}

export function useDialog() {
    const context = useContext(DialogContext)
    if (!context) {
        throw new Error('useDialog must be used within a DialogProvider')
    }
    return context
}
