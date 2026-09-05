/**
 * Moly Agent - TypeScript ↔ Go Bridge
 * Connects extension to Go backend systems:
 * - SafetyChecker (crisis/illegal detection)
 * - ConstitutionEvaluator (ethics assessment)
 * - ModeTransitionEngine (relationship analysis)
 * - QuestionAgent (contextual questions)
 */

const GO_BACKEND_URL = 'http://127.0.0.1:11436';

export interface SafetyCheckResult {
  alert_type: 'crisis' | 'illegal' | 'none';
  severity: 'immediate' | 'high' | 'warning';
  title: string;
  message: string;
  indicators: string[];
  resources: CrisisResource[];
  recommendations: string[];
}

export interface CrisisResource {
  name: string;
  description: string;
  number: string;
  url: string;
  region: string;
}

export interface ConstitutionViolation {
  principle_id: string;
  principle: string;
  severity: 'critical' | 'high' | 'medium';
  description: string;
  reasoning: string;
}

export interface ConstitutionalAnalysis {
  analyzed_action: string;
  violations: ConstitutionViolation[];
  aligned_principles: string[];
  overall_risk_level: string;
  critical_concerns: string[];
  recommendations: string[];
  is_constitutional: boolean;
}

export interface ModeShiftAnalysis {
  relationship_modes: {
    current: string;
    proposed: string;
  };
  phase: string;
  risk_level: number;
  indicators: {
    clear_indicators: string[];
    potential_concerns: string[];
  };
  mitigation_strategies: string[];
  questions_for_reflection: string[];
}

export interface QuestionGeneratorResult {
  questions: string[];
  context: string;
  reasoning: string;
}

export interface CommunicationPrinciple {
  id: string;
  name: string;
  severity: 'critical' | 'high' | 'medium';
  description: string;
  questions: string[];
}

class MolyAgent {
  private isBackendAvailable: boolean = false;
  private checkBackendPromise: Promise<boolean> | null = null;

  constructor() {
    this.checkBackendAvailability();
  }

  /**
   * Check if Go backend is available
   */
  private async checkBackendAvailability(): Promise<boolean> {
    if (this.checkBackendPromise) {
      return this.checkBackendPromise;
    }

    this.checkBackendPromise = (async () => {
      try {
        const response = await fetch(`${GO_BACKEND_URL}/api/status`, {
          method: 'GET',
          headers: { 'Content-Type': 'application/json' },
        });
        this.isBackendAvailable = response.ok;
        console.log(`[MolyAgent] Backend available: ${this.isBackendAvailable}`);
        return this.isBackendAvailable;
      } catch (error) {
        console.warn('[MolyAgent] Backend not available:', error);
        this.isBackendAvailable = false;
        return false;
      }
    })();

    return this.checkBackendPromise;
  }

  /**
   * Ensure backend is available before making requests
   */
  private async ensureBackend(): Promise<void> {
    if (!this.isBackendAvailable) {
      const available = await this.checkBackendAvailability();
      if (!available) {
        throw new Error(
          'Moly backend is not running. Start the Go server to enable advanced features.'
        );
      }
    }
  }

  /**
   * Check message for safety issues (crisis/illegal language)
   */
  async checkSafety(message: string): Promise<SafetyCheckResult> {
    try {
      await this.ensureBackend();

      const response = await fetch(`${GO_BACKEND_URL}/api/check-safety`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message }),
      });

      if (!response.ok) {
        throw new Error(`Safety check failed: ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      console.error('[MolyAgent] Safety check error:', error);
      throw error;
    }
  }

  /**
   * Evaluate message against 10 ethical principles
   */
  async evaluateConstitution(message: string): Promise<ConstitutionalAnalysis> {
    try {
      await this.ensureBackend();

      const response = await fetch(`${GO_BACKEND_URL}/api/evaluate-constitution`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message }),
      });

      if (!response.ok) {
        throw new Error(`Constitution evaluation failed: ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      console.error('[MolyAgent] Constitution evaluation error:', error);
      throw error;
    }
  }

  /**
   * Analyze relationship mode shift (e.g., Professional → Romantic)
   */
  async analyzeModeShift(
    currentMode: string,
    proposedMode: string,
    context: string
  ): Promise<ModeShiftAnalysis> {
    try {
      await this.ensureBackend();

      const response = await fetch(`${GO_BACKEND_URL}/api/analyze-mode-shift`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          current_mode: currentMode,
          proposed_mode: proposedMode,
          context,
        }),
      });

      if (!response.ok) {
        throw new Error(`Mode shift analysis failed: ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      console.error('[MolyAgent] Mode shift analysis error:', error);
      throw error;
    }
  }

  /**
   * Generate contextual questions based on contact info
   */
  async generateQuestions(contactName: string, context: string): Promise<QuestionGeneratorResult> {
    try {
      await this.ensureBackend();

      const response = await fetch(`${GO_BACKEND_URL}/api/generate-questions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          contact_name: contactName,
          context,
        }),
      });

      if (!response.ok) {
        throw new Error(`Question generation failed: ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      console.error('[MolyAgent] Question generation error:', error);
      throw error;
    }
  }

  /**
   * Get all communication principles
   */
  async getPrinciples(): Promise<CommunicationPrinciple[]> {
    try {
      await this.ensureBackend();

      const response = await fetch(`${GO_BACKEND_URL}/api/constitution-principles`, {
        method: 'GET',
        headers: { 'Content-Type': 'application/json' },
      });

      if (!response.ok) {
        throw new Error(`Failed to get principles: ${response.statusText}`);
      }

      const data = await response.json();
      return data.principles || [];
    } catch (error) {
      console.error('[MolyAgent] Get principles error:', error);
      throw error;
    }
  }

  /**
   * Run complete analysis pipeline (safety → constitution → questions)
   */
  async analyzeMessage(
    message: string,
    contactName?: string,
    context?: string
  ): Promise<{
    safety: SafetyCheckResult;
    constitution?: ConstitutionalAnalysis;
    questions?: QuestionGeneratorResult;
  }> {
    const results: any = {};

    try {
      // Always run safety check first
      results.safety = await this.checkSafety(message);

      // If not a crisis, run other checks
      if (results.safety.alert_type === 'none') {
        try {
          results.constitution = await this.evaluateConstitution(message);
        } catch (error) {
          console.warn('[MolyAgent] Constitution check skipped:', error);
        }

        if (contactName && context) {
          try {
            results.questions = await this.generateQuestions(contactName, context);
          } catch (error) {
            console.warn('[MolyAgent] Question generation skipped:', error);
          }
        }
      }
    } catch (error) {
      console.error('[MolyAgent] Analysis pipeline error:', error);
      throw error;
    }

    return results;
  }

  /**
   * Check if backend is available (non-throwing)
   */
  async isAvailable(): Promise<boolean> {
    return this.checkBackendAvailability();
  }
}

// Singleton instance
let agent: MolyAgent | null = null;

export function getMolyAgent(): MolyAgent {
  if (!agent) {
    agent = new MolyAgent();
  }
  return agent;
}

export default MolyAgent;
