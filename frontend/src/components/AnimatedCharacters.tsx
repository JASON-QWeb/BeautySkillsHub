import { useState, useEffect, useRef } from 'react'

interface PupilProps {
    size?: number
    maxDistance?: number
    color?: string
    forceLookX?: number
    forceLookY?: number
}

function Pupil({ size = 12, maxDistance = 5, color = '#2D2D2D', forceLookX, forceLookY }: PupilProps) {
    const [mouseX, setMouseX] = useState(0)
    const [mouseY, setMouseY] = useState(0)
    const ref = useRef<HTMLDivElement>(null)

    useEffect(() => {
        const onMove = (e: MouseEvent) => { setMouseX(e.clientX); setMouseY(e.clientY) }
        window.addEventListener('mousemove', onMove)
        return () => window.removeEventListener('mousemove', onMove)
    }, [])

    const pos = (() => {
        if (!ref.current) return { x: 0, y: 0 }
        if (forceLookX !== undefined && forceLookY !== undefined) return { x: forceLookX, y: forceLookY }
        const rect = ref.current.getBoundingClientRect()
        const dx = mouseX - (rect.left + rect.width / 2)
        const dy = mouseY - (rect.top + rect.height / 2)
        const dist = Math.min(Math.sqrt(dx * dx + dy * dy), maxDistance)
        const angle = Math.atan2(dy, dx)
        return { x: Math.cos(angle) * dist, y: Math.sin(angle) * dist }
    })()

    return (
        <div
            ref={ref}
            style={{
                width: size, height: size, borderRadius: '50%',
                backgroundColor: color,
                transform: `translate(${pos.x}px, ${pos.y}px)`,
                transition: 'transform 0.1s ease-out',
            }}
        />
    )
}

interface EyeBallProps {
    size?: number
    pupilSize?: number
    maxDistance?: number
    eyeColor?: string
    pupilColor?: string
    isBlinking?: boolean
    forceLookX?: number
    forceLookY?: number
}

function EyeBall({
    size = 48, pupilSize = 16, maxDistance = 10,
    eyeColor = 'white', pupilColor = '#2D2D2D',
    isBlinking = false, forceLookX, forceLookY,
}: EyeBallProps) {
    const [mouseX, setMouseX] = useState(0)
    const [mouseY, setMouseY] = useState(0)
    const ref = useRef<HTMLDivElement>(null)

    useEffect(() => {
        const onMove = (e: MouseEvent) => { setMouseX(e.clientX); setMouseY(e.clientY) }
        window.addEventListener('mousemove', onMove)
        return () => window.removeEventListener('mousemove', onMove)
    }, [])

    const pos = (() => {
        if (!ref.current) return { x: 0, y: 0 }
        if (forceLookX !== undefined && forceLookY !== undefined) return { x: forceLookX, y: forceLookY }
        const rect = ref.current.getBoundingClientRect()
        const dx = mouseX - (rect.left + rect.width / 2)
        const dy = mouseY - (rect.top + rect.height / 2)
        const dist = Math.min(Math.sqrt(dx * dx + dy * dy), maxDistance)
        const angle = Math.atan2(dy, dx)
        return { x: Math.cos(angle) * dist, y: Math.sin(angle) * dist }
    })()

    return (
        <div
            ref={ref}
            style={{
                width: size, height: isBlinking ? 2 : size, borderRadius: '50%',
                backgroundColor: eyeColor, overflow: 'hidden',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                transition: 'height 0.15s ease',
            }}
        >
            {!isBlinking && (
                <div
                    style={{
                        width: pupilSize, height: pupilSize, borderRadius: '50%',
                        backgroundColor: pupilColor,
                        transform: `translate(${pos.x}px, ${pos.y}px)`,
                        transition: 'transform 0.1s ease-out',
                    }}
                />
            )}
        </div>
    )
}

interface AnimatedCharactersProps {
    isTyping?: boolean
    showPassword?: boolean
    passwordLength?: number
}

export default function AnimatedCharacters({
    isTyping = false,
    showPassword = false,
    passwordLength = 0,
}: AnimatedCharactersProps) {
    const [mouseX, setMouseX] = useState(0)
    const [mouseY, setMouseY] = useState(0)
    const [purpleBlink, setPurpleBlink] = useState(false)
    const [blackBlink, setBlackBlink] = useState(false)
    const [lookingAtEachOther, setLookingAtEachOther] = useState(false)
    const [purplePeeking, setPurplePeeking] = useState(false)

    const purpleRef = useRef<HTMLDivElement>(null)
    const blackRef = useRef<HTMLDivElement>(null)
    const yellowRef = useRef<HTMLDivElement>(null)
    const orangeRef = useRef<HTMLDivElement>(null)

    useEffect(() => {
        const onMove = (e: MouseEvent) => { setMouseX(e.clientX); setMouseY(e.clientY) }
        window.addEventListener('mousemove', onMove)
        return () => window.removeEventListener('mousemove', onMove)
    }, [])

    // Blinking effects
    useEffect(() => {
        const schedule = (setter: (v: boolean) => void) => {
            const id = setTimeout(() => {
                setter(true)
                setTimeout(() => { setter(false); schedule(setter) }, 150)
            }, Math.random() * 4000 + 3000)
            return id
        }
        const t1 = schedule(setPurpleBlink)
        const t2 = schedule(setBlackBlink)
        return () => { clearTimeout(t1); clearTimeout(t2) }
    }, [])

    // Look at each other when typing starts
    useEffect(() => {
        if (isTyping) {
            setLookingAtEachOther(true)
            const t = setTimeout(() => setLookingAtEachOther(false), 800)
            return () => clearTimeout(t)
        }
        setLookingAtEachOther(false)
    }, [isTyping])

    // Purple peeks when password visible
    useEffect(() => {
        if (passwordLength > 0 && showPassword) {
            const t = setTimeout(() => {
                setPurplePeeking(true)
                setTimeout(() => setPurplePeeking(false), 800)
            }, Math.random() * 3000 + 2000)
            return () => clearTimeout(t)
        }
        setPurplePeeking(false)
    }, [passwordLength, showPassword, purplePeeking])

    const calcPos = (ref: React.RefObject<HTMLDivElement | null>) => {
        if (!ref.current) return { faceX: 0, faceY: 0, bodySkew: 0 }
        const rect = ref.current.getBoundingClientRect()
        const dx = mouseX - (rect.left + rect.width / 2)
        const dy = mouseY - (rect.top + rect.height / 3)
        return {
            faceX: Math.max(-15, Math.min(15, dx / 20)),
            faceY: Math.max(-10, Math.min(10, dy / 30)),
            bodySkew: Math.max(-6, Math.min(6, -dx / 120)),
        }
    }

    const pp = calcPos(purpleRef)
    const bp = calcPos(blackRef)
    const yp = calcPos(yellowRef)
    const op = calcPos(orangeRef)

    const isHiding = passwordLength > 0 && !showPassword
    const isPwdVisible = passwordLength > 0 && showPassword

    return (
        <div style={{ position: 'relative', width: 320, height: 230 }}>
            {/* Purple character */}
            <div
                ref={purpleRef}
                className="char-body"
                style={{
                    position: 'absolute', bottom: 0, left: 40,
                    width: 105, height: (isTyping || isHiding) ? 260 : 230,
                    backgroundColor: '#6C3FF5', borderRadius: '10px 10px 0 0',
                    zIndex: 1, transformOrigin: 'bottom center',
                    transform: isPwdVisible
                        ? 'skewX(0deg)'
                        : (isTyping || isHiding)
                            ? `skewX(${(pp.bodySkew || 0) - 12}deg) translateX(24px)`
                            : `skewX(${pp.bodySkew || 0}deg)`,
                }}
            >
                <div
                    className="char-eyes"
                    style={{
                        position: 'absolute', display: 'flex', gap: 18,
                        left: isPwdVisible ? 12 : lookingAtEachOther ? 32 : 26 + pp.faceX,
                        top: isPwdVisible ? 20 : lookingAtEachOther ? 38 : 24 + pp.faceY,
                    }}
                >
                    <EyeBall size={14} pupilSize={6} maxDistance={4} isBlinking={purpleBlink}
                        forceLookX={isPwdVisible ? (purplePeeking ? 4 : -4) : lookingAtEachOther ? 3 : undefined}
                        forceLookY={isPwdVisible ? (purplePeeking ? 5 : -4) : lookingAtEachOther ? 4 : undefined}
                    />
                    <EyeBall size={14} pupilSize={6} maxDistance={4} isBlinking={purpleBlink}
                        forceLookX={isPwdVisible ? (purplePeeking ? 4 : -4) : lookingAtEachOther ? 3 : undefined}
                        forceLookY={isPwdVisible ? (purplePeeking ? 5 : -4) : lookingAtEachOther ? 4 : undefined}
                    />
                </div>
            </div>

            {/* Black character */}
            <div
                ref={blackRef}
                className="char-body"
                style={{
                    position: 'absolute', bottom: 0, left: 140,
                    width: 70, height: 180,
                    backgroundColor: '#2D2D2D', borderRadius: '8px 8px 0 0',
                    zIndex: 2, transformOrigin: 'bottom center',
                    transform: isPwdVisible
                        ? 'skewX(0deg)'
                        : lookingAtEachOther
                            ? `skewX(${(bp.bodySkew || 0) * 1.5 + 10}deg) translateX(12px)`
                            : (isTyping || isHiding)
                                ? `skewX(${(bp.bodySkew || 0) * 1.5}deg)`
                                : `skewX(${bp.bodySkew || 0}deg)`,
                }}
            >
                <div
                    className="char-eyes"
                    style={{
                        position: 'absolute', display: 'flex', gap: 14,
                        left: isPwdVisible ? 6 : lookingAtEachOther ? 18 : 15 + bp.faceX,
                        top: isPwdVisible ? 16 : lookingAtEachOther ? 8 : 18 + bp.faceY,
                    }}
                >
                    <EyeBall size={12} pupilSize={5} maxDistance={3} isBlinking={blackBlink}
                        forceLookX={isPwdVisible ? -4 : lookingAtEachOther ? 0 : undefined}
                        forceLookY={isPwdVisible ? -4 : lookingAtEachOther ? -4 : undefined}
                    />
                    <EyeBall size={12} pupilSize={5} maxDistance={3} isBlinking={blackBlink}
                        forceLookX={isPwdVisible ? -4 : lookingAtEachOther ? 0 : undefined}
                        forceLookY={isPwdVisible ? -4 : lookingAtEachOther ? -4 : undefined}
                    />
                </div>
            </div>

            {/* Orange semi-circle */}
            <div
                ref={orangeRef}
                className="char-body"
                style={{
                    position: 'absolute', bottom: 0, left: 0,
                    width: 140, height: 116,
                    backgroundColor: '#FF9B6B', borderRadius: '70px 70px 0 0',
                    zIndex: 3, transformOrigin: 'bottom center',
                    transform: isPwdVisible ? 'skewX(0deg)' : `skewX(${op.bodySkew || 0}deg)`,
                }}
            >
                <div
                    className="char-eyes"
                    style={{
                        position: 'absolute', display: 'flex', gap: 18,
                        left: isPwdVisible ? 29 : 48 + op.faceX,
                        top: isPwdVisible ? 50 : 52 + op.faceY,
                    }}
                >
                    <Pupil size={10} maxDistance={4} forceLookX={isPwdVisible ? -5 : undefined} forceLookY={isPwdVisible ? -4 : undefined} />
                    <Pupil size={10} maxDistance={4} forceLookX={isPwdVisible ? -5 : undefined} forceLookY={isPwdVisible ? -4 : undefined} />
                </div>
            </div>

            {/* Yellow character */}
            <div
                ref={yellowRef}
                className="char-body"
                style={{
                    position: 'absolute', bottom: 0, left: 180,
                    width: 82, height: 134,
                    backgroundColor: '#E8D754', borderRadius: '41px 41px 0 0',
                    zIndex: 4, transformOrigin: 'bottom center',
                    transform: isPwdVisible ? 'skewX(0deg)' : `skewX(${yp.bodySkew || 0}deg)`,
                }}
            >
                <div
                    className="char-eyes"
                    style={{
                        position: 'absolute', display: 'flex', gap: 14,
                        left: isPwdVisible ? 12 : 30 + yp.faceX,
                        top: isPwdVisible ? 20 : 24 + yp.faceY,
                    }}
                >
                    <Pupil size={10} maxDistance={4} forceLookX={isPwdVisible ? -5 : undefined} forceLookY={isPwdVisible ? -4 : undefined} />
                    <Pupil size={10} maxDistance={4} forceLookX={isPwdVisible ? -5 : undefined} forceLookY={isPwdVisible ? -4 : undefined} />
                </div>
                {/* Mouth */}
                <div
                    style={{
                        position: 'absolute', width: 46, height: 3,
                        backgroundColor: '#2D2D2D', borderRadius: 2,
                        left: isPwdVisible ? 6 : 24 + yp.faceX,
                        top: isPwdVisible ? 52 : 52 + yp.faceY,
                        transition: 'left 0.2s ease-out, top 0.2s ease-out',
                    }}
                />
            </div>
        </div>
    )
}
