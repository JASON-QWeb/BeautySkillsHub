import { useState } from 'react'
import { AIChatCharacter } from './AIChatCharacter'

interface SkillsIntroModalProps {
    isOpen: boolean
    onClose: () => void
}

export function SkillsIntroModal({ isOpen, onClose }: SkillsIntroModalProps) {
    const [currentSlide, setCurrentSlide] = useState(0)
    const slides = [
        '/images/d1.webp',
        '/images/d2.webp',
        '/images/d3.webp',
        '/images/d4.webp',
    ]

    if (!isOpen) return null

    const handleNext = () => {
        if (currentSlide < slides.length - 1) {
            setCurrentSlide(prev => prev + 1)
        }
    }

    const handlePrev = () => {
        if (currentSlide > 0) {
            setCurrentSlide(prev => prev - 1)
        }
    }

    return (
        <div className="skills-intro-modal-overlay" onClick={onClose}>
            <div className="skills-intro-modal-box" onClick={e => e.stopPropagation()}>
                <button className="skills-intro-close" onClick={onClose} aria-label="Close">
                    ✕
                </button>

                <AIChatCharacter 
                    className="modal-corner-character" 
                    isOpen={false} 
                    isTyping={false} 
                />

                <div className="skills-intro-slides-container">
                    {slides.map((slide, index) => (
                        <img
                            key={slide}
                            src={slide}
                            alt={`Slide ${index + 1}`}
                            className={`skills-intro-modal-image ${index === currentSlide ? 'active' : ''}`}
                        />
                    ))}
                </div>
                
                {currentSlide > 0 && (
                    <button className="skills-intro-arrow prev" onClick={handlePrev}>
                        ‹
                    </button>
                )}

                {currentSlide < slides.length - 1 && (
                    <button className="skills-intro-arrow next" onClick={handleNext}>
                        ›
                    </button>
                )}
            </div>
        </div>
    )
}
