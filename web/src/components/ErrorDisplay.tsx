import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { ChevronDown, ChevronUp, Copy, Check } from 'lucide-react'

interface ErrorDisplayProps {
  error: string
  compact?: boolean
  className?: string
}

export function ErrorDisplay({ error, compact = false, className = '' }: ErrorDisplayProps) {
  const [isExpanded, setIsExpanded] = useState(false)
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(error)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy:', err)
    }
  }

  // Truncate long errors for compact view
  const truncatedError = error.length > 150 ? error.substring(0, 150) + '...' : error
  const displayError = compact && !isExpanded ? truncatedError : error
  const shouldShowToggle = compact && error.length > 150

  return (
    <div className={`p-3 bg-red-50 border border-red-200 rounded-lg ${className}`}>
      <div className="flex items-start justify-between mb-2">
        <div className="font-medium text-red-800 text-sm">Error Details</div>
        <div className="flex items-center space-x-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleCopy}
            className="h-6 w-6 p-0 text-red-600 hover:text-red-800"
            title="Copy error message"
          >
            {copied ? (
              <Check className="h-3 w-3" />
            ) : (
              <Copy className="h-3 w-3" />
            )}
          </Button>
          {shouldShowToggle && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setIsExpanded(!isExpanded)}
              className="h-6 w-6 p-0 text-red-600 hover:text-red-800"
              title={isExpanded ? "Show less" : "Show more"}
            >
              {isExpanded ? (
                <ChevronUp className="h-3 w-3" />
              ) : (
                <ChevronDown className="h-3 w-3" />
              )}
            </Button>
          )}
        </div>
      </div>
      
      <div className={`text-xs text-red-800 break-words whitespace-pre-wrap leading-relaxed ${
        compact ? 'max-h-32' : 'max-h-40'
      } overflow-y-auto`}>
        {displayError}
      </div>
      
      {copied && (
        <div className="mt-2 text-xs text-green-600 font-medium">
          ✓ Copied to clipboard
        </div>
      )}
    </div>
  )
}