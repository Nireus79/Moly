# Moly Success Metrics & Analytics

**Date**: September 6, 2026  
**Status**: Metrics Reference  
**Version**: 1.0

---

## North Star Metrics

### 1. User Autonomy & Authenticity

**Metric**: Suggestion Pick Rate
- **What**: % of generated suggestions user chooses to send
- **Good**: > 70% (suggestions feel right)
- **Target**: 80%+
- **How to measure**: `COUNT(suggestions_chosen) / COUNT(suggestions_generated)`

**Metric**: Modification Rate (Quality Signal)
- **What**: % of suggestions user edits before sending
- **Good**: 40-60% (user adds personality, not using as-is)
- **Bad**: < 20% (suggestions too generic) or > 80% (too off)
- **Target**: 50%
- **How to measure**: `COUNT(suggestions_modified) / COUNT(suggestions_chosen)`

**Metric**: Direct Copy Rate (Authenticity)
- **What**: % of suggestions copied directly without editing
- **Signals**: Authenticity (low = good, suggestion matches user style)
- **Target**: 40-50% direct copy
- **How to measure**: `COUNT(direct_copy) / COUNT(suggestions_chosen)`

---

### 2. Effectiveness & Impact

**Metric**: Positive Response Rate
- **What**: % of sent messages that get positive contact response (inferred)
- **Good**: > 70% (messages are working)
- **How measured**: User feedback, conversation continuation, re-engagement
- **Target**: 75%+

**Metric**: Conversation Continuation Rate
- **What**: % of conversations with follow-up messages
- **Good**: > 60% (relationships deepening)
- **Target**: 70%+
- **How to measure**: `COUNT(convos_with_follow_up) / COUNT(total_convos)`

**Metric**: Contact Re-engagement
- **What**: User starts new conversation with same contact (weeks later)
- **Good**: High (Moly helps maintain relationships)
- **Target**: 80% of contacts contacted again within 3 months

---

### 3. Learning & Improvement

**Metric**: Personalization Quality
- **What**: How well suggestions match user's authentic style
- **Signal**: Modification rate decreasing over time (better matches)
- **Target**: 10% improvement per month
- **How to measure**: 
  - Month 1: 60% modification rate
  - Month 2: 54% modification rate
  - Month 3: 48% modification rate

**Metric**: User Confidence Trend
- **What**: User's perceived confidence in their communication
- **Signals**:
  - More direct copy (not editing)
  - Faster suggestion acceptance
  - Less time per conversation
- **Target**: +15% confidence per quarter

**Metric**: Skill Development
- **What**: User's actual communication improving
- **Signals**:
  - Higher positive response rate
  - Longer conversations
  - Deeper relationships
- **Target**: +20% effectiveness per quarter

---

## Product Metrics

### Engagement

| Metric | Good | Target | Method |
|--------|------|--------|--------|
| DAU (Daily Active Users) | > 100k | 1M+ | Count daily unique users |
| WAU (Weekly Active Users) | > 500k | 5M+ | Count weekly unique |
| MAU (Monthly Active Users) | > 1M | 10M+ | Count monthly unique |
| Retention Day 7 | > 40% | 60%+ | (users active day 7) / (users created) |
| Retention Day 30 | > 25% | 40%+ | (users active day 30) / (users created) |
| Churn Rate | < 10%/month | < 5%/month | (lost users) / (prior users) |

### Usage

| Metric | Good | Target | Method |
|--------|------|--------|--------|
| Avg Messages per User/Day | > 5 | 10+ | Sum(messages) / DAU |
| Conversations per User | > 50 | 100+ | Sum(convos) / total_users |
| Contacts per User | > 10 | 20+ | Sum(contacts) / total_users |
| Session Length | > 5 min | 10+ min | Average(duration) |
| Sessions per User/Week | > 3 | 7+ | Count(sessions) / WAU |

---

## Quality Metrics

### Suggestion Quality

**User Satisfaction** (Survey-based):
```
Question: "How often do our suggestions feel like you?"
- Always (90-100%): 4 points
- Usually (70-89%): 3 points
- Sometimes (50-69%): 2 points
- Rarely (< 50%): 1 point

Target: Average 3.5+ points
```

**Suggestion Accuracy**:
- Tone accuracy: Suggestion matches requested tone 90%+
- Length accuracy: Suggestion matches user preference 85%+
- Style accuracy: Suggestion feels authentic 80%+

### Safety Quality

**Safety Alert Accuracy**:
- True Positives: Correctly identified risky messages
- False Positives: Flagged safe messages
- Target: > 95% accuracy, < 5% false positive rate

**User Override Rate**:
- % of safety warnings user overrides to proceed
- Signal: Should be low (< 20%) — education working
- If high: Change approach

---

## Learning & Intelligence Metrics

### User Learning

| Metric | Target | Method |
|--------|--------|--------|
| Profile Completeness | > 90% | (filled fields) / (total fields) |
| Contact Details Quality | > 80% | (contacts with > 5 attributes) / (total) |
| User Personality Accuracy | > 75% | (inferred traits match self-report) |
| Behavior Prediction Accuracy | > 80% | (predicted choices vs actual) |

### Agent Performance

| Metric | Target | Method |
|--------|--------|--------|
| Suggestion Generation Latency | < 1s p50 | Histogram of response times |
| Question Generation Latency | < 500ms p50 | Histogram |
| Safety Check Latency | < 200ms p50 | Histogram |
| Agent Success Rate | > 99% | (completed / started) |
| Agent Error Rate | < 1% | (errors / requests) |

---

## Business Metrics

### Monetization (Future)

**If subscription model**:
- Conversion Rate: % free users → paid
- ARPU: Average Revenue Per User
- LTV: Lifetime Value
- CAC: Customer Acquisition Cost
- LTV/CAC Ratio: Target > 3:1

**If freemium**:
- Free-to-paid conversion: Target 5-10%
- Trial completion: Target > 60%
- Upsell rate: Target > 30%

### Growth

- **Growth Rate**: % new users per month
- **Viral Coefficient**: k-factor (word-of-mouth multiplier)
- **Referral Rate**: % users who refer others

---

## Ethical Metrics

### Privacy

**Metric**: Data Access Requests
- Count of "I want my data" requests
- Target: Handle 100% within 30 days

**Metric**: Privacy Complaints
- Count of privacy concerns raised
- Target: < 1 per 100,000 users

**Metric**: Data Retention Compliance
- % of old data properly deleted per policy
- Target: 100%

### Autonomy

**Metric**: User Override Rate
- % of Moly's safety warnings user overrides
- Target: 15-20% (user makes conscious choice)
- If < 5%: User trusting blindly (bad)
- If > 40%: Warnings too aggressive (bad)

**Metric**: User Customization
- % of users customizing their About Me
- % of users customizing contact notes
- Target: > 80%

---

## Monitoring Dashboard

**Real-time**:
- API latency (p50, p95, p99)
- Error rates by endpoint
- Agent processing times
- Database connection pool
- Cache hit rate

**Daily**:
- DAU, WAU, MAU
- Retention metrics
- Suggestion pick rate
- Safety alert rate
- Top errors

**Weekly**:
- Cohort analysis (user retention by signup week)
- Feature adoption
- Quality metrics
- Personalization improvements

**Monthly**:
- Growth rate
- LTV/CAC
- User satisfaction survey
- Behavioral trends
- Risk pattern updates

---

## Alerts & Thresholds

**Critical** (page on-call):
- Error rate > 5% for 5 min
- API latency p95 > 5s for 5 min
- Database unavailable
- Agent failure rate > 1%

**Warning** (investigate within 4 hours):
- DAU down > 20% vs yesterday
- Retention dropping > 5%
- Suggestion pick rate down > 10%
- False positive rate > 10%

**Info** (daily review):
- New user growth trend
- Feature adoption
- Cohort performance

---

## Success Definition

**Moly is successful when:**

1. **Users trust Moly** — Pick rate > 70%, satisfaction > 3.5/4
2. **Users improve** — Response rate > 70%, confidence increasing
3. **Relationships deepen** — Conversation continuation > 60%, re-engagement > 80%
4. **Ethics work** — User overrides 15-20%, no privacy complaints
5. **Scale works** — 1M+ users, < 5% error rate, < 1s latency
6. **Business works** — Retention > 40% day 7, LTV/CAC > 3

---

## OKR Example (Q4 2026)

**Objective**: Moly becomes trusted communication thinking partner

**Key Results**:
1. Reach 500K MAU (from 100K)
2. Achieve 60% day-7 retention (from 40%)
3. Get suggestion pick rate to 75% (from 65%)
4. Reach 75% user satisfaction (from 65%)
5. Reduce false positive safety rate to 5% (from 8%)

**Initiatives**:
- Improve personalization algorithm (→ higher pick rate)
- Better onboarding (→ higher retention)
- Refine safety thresholds (→ lower false positives)
- Marketing push (→ 500K MAU)

---

## Measurement Tools

- **Metrics Collection**: Prometheus + Grafana
- **User Analytics**: Mixpanel or Amplitude
- **Surveys**: Typeform or Qualtrics
- **A/B Testing**: Split.io or Statsig
- **Error Tracking**: Sentry or Rollbar
- **Logs**: ELK Stack or Datadog

---

Success metrics ensure Moly is actually helping users improve their authentic communication.
