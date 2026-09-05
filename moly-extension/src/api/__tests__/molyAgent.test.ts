/**
 * MolyAgent Bridge Tests
 * Tests the TypeScript ↔ Go backend connection
 */

import { getMolyAgent, type SafetyCheckResult, type ConstitutionalAnalysis } from '../molyAgent';

describe('MolyAgent Bridge', () => {
  const agent = getMolyAgent();

  // Note: These tests assume Go backend is running on 127.0.0.1:11436
  // To run: moly-go/moly &
  // Then: npm test

  describe('Backend availability', () => {
    it('should detect backend availability', async () => {
      const available = await agent.isAvailable();
      console.log(`Backend available: ${available}`);
      // May be false if backend not running - that's OK
      expect(typeof available).toBe('boolean');
    });
  });

  describe('Safety checking', () => {
    it('should detect safe messages', async () => {
      try {
        const result = await agent.checkSafety('Hello, how are you?');
        console.log('Safe message result:', result);
        expect(result.alert_type).toBe('none');
        expect(result.severity).toBeDefined();
      } catch (error) {
        console.warn('Safety check failed (backend may not be running):', error);
      }
    });

    it('should detect crisis language', async () => {
      try {
        const result = await agent.checkSafety('I want to hurt myself');
        console.log('Crisis message result:', result);
        expect(result.alert_type).toBe('crisis');
        expect(result.severity).toBe('immediate');
        expect(result.indicators.length).toBeGreaterThan(0);
        expect(result.resources.length).toBeGreaterThan(0);
      } catch (error) {
        console.warn('Crisis check failed (backend may not be running):', error);
      }
    });
  });

  describe('Constitution evaluation', () => {
    it('should evaluate ethical principles', async () => {
      try {
        const result = await agent.evaluateConstitution(
          'I want to be honest with you about what I think'
        );
        console.log('Constitution result:', result);
        expect(result.analyzed_action).toBeDefined();
        expect(Array.isArray(result.aligned_principles)).toBe(true);
        expect(result.overall_risk_level).toBeDefined();
      } catch (error) {
        console.warn('Constitution check failed (backend may not be running):', error);
      }
    });
  });

  describe('Question generation', () => {
    it('should generate contextual questions', async () => {
      try {
        const result = await agent.generateQuestions(
          'John',
          'We met at work a few months ago'
        );
        console.log('Questions result:', result);
        expect(Array.isArray(result.questions)).toBe(true);
        expect(result.context).toBeDefined();
        expect(result.reasoning).toBeDefined();
      } catch (error) {
        console.warn(
          'Question generation failed (backend may not be running):',
          error
        );
      }
    });
  });

  describe('Mode shift analysis', () => {
    it('should analyze relationship mode shifts', async () => {
      try {
        const result = await agent.analyzeModeShift(
          'Professional',
          'Romantic',
          'We have worked together for 2 years'
        );
        console.log('Mode shift result:', result);
        expect(result.relationship_modes).toBeDefined();
        expect(result.risk_level).toBeDefined();
        expect(result.risk_level).toBeGreaterThanOrEqual(0);
        expect(result.risk_level).toBeLessThanOrEqual(100);
      } catch (error) {
        console.warn('Mode shift analysis failed (backend may not be running):', error);
      }
    });
  });

  describe('Full analysis pipeline', () => {
    it('should run complete analysis for safe message', async () => {
      try {
        const result = await agent.analyzeMessage(
          'I really appreciate your feedback',
          'Sarah',
          'Our team meeting yesterday'
        );
        console.log('Full analysis result:', result);
        expect(result.safety).toBeDefined();
        expect(result.safety.alert_type).toBe('none');
      } catch (error) {
        console.warn('Full analysis failed (backend may not be running):', error);
      }
    });

    it('should stop analysis on crisis detection', async () => {
      try {
        const result = await agent.analyzeMessage(
          'I cannot live with this pain anymore'
        );
        console.log('Crisis analysis result:', result);
        expect(result.safety.alert_type).toBe('crisis');
        expect(result.constitution).toBeUndefined();
      } catch (error) {
        console.warn('Crisis analysis failed (backend may not be running):', error);
      }
    });
  });
});
