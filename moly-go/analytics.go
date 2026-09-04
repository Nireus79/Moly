package main

import (
	"database/sql"
	"fmt"
	"sort"
	"time"
)

type Analytics struct {
	db *Database
}

type ContactStats struct {
	ContactID        int    `json:"contact_id"`
	Name             string `json:"name"`
	Platform         string `json:"platform"`
	InteractionCount int    `json:"interaction_count"`
	LastInteraction  string `json:"last_interaction"`
	TopTopic         string `json:"top_topic"`
}

type TopicStats struct {
	Topic            string `json:"topic"`
	ContactCount     int    `json:"contact_count"`
	InteractionCount int    `json:"interaction_count"`
	AvgSentiment     string `json:"avg_sentiment"`
}

type ToneStats struct {
	Period        string `json:"period"`
	Sentiment     string `json:"sentiment"`
	Count         int    `json:"count"`
	Percentage    float64 `json:"percentage"`
}

type SummaryStats struct {
	TotalContacts      int       `json:"total_contacts"`
	TotalInteractions  int       `json:"total_interactions"`
	MostFrequentContact string  `json:"most_frequent_contact"`
	MostCommonTopic    string  `json:"most_common_topic"`
	DominantTone       string  `json:"dominant_tone"`
	AverageTopics      float64 `json:"avg_topics_per_interaction"`
	NewContactsThisWeek int    `json:"new_contacts_this_week"`
}

type CommunicationPattern struct {
	Contact     string  `json:"contact"`
	Frequency   int     `json:"frequency"`
	AvgTone     string  `json:"avg_tone"`
	TopTopics   []string `json:"top_topics"`
	LastContact string  `json:"last_contact"`
}

func NewAnalytics(db *Database) *Analytics {
	return &Analytics{db: db}
}

func (a *Analytics) GetContactStats() ([]ContactStats, error) {
	rows, err := a.db.conn.Query(`
		SELECT c.id, c.name, c.platform, c.interaction_count, c.last_interaction
		FROM contacts c
		ORDER BY c.interaction_count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []ContactStats
	for rows.Next() {
		var s ContactStats
		var lastInt sql.NullTime

		err := rows.Scan(&s.ContactID, &s.Name, &s.Platform, &s.InteractionCount, &lastInt)
		if err != nil {
			return nil, err
		}

		if lastInt.Valid {
			s.LastInteraction = lastInt.Time.Format("2006-01-02 15:04")
		} else {
			s.LastInteraction = "Never"
		}

		// Get top topic for this contact
		var topTopic sql.NullString
		a.db.conn.QueryRow(`
			SELECT topic FROM interactions
			WHERE contact_id = ? AND topic != ''
			GROUP BY topic ORDER BY COUNT(*) DESC LIMIT 1
		`, s.ContactID).Scan(&topTopic)

		if topTopic.Valid {
			s.TopTopic = topTopic.String
		} else {
			s.TopTopic = "General"
		}

		stats = append(stats, s)
	}

	return stats, nil
}

func (a *Analytics) GetTopicStats() ([]TopicStats, error) {
	rows, err := a.db.conn.Query(`
		SELECT topic, COUNT(DISTINCT contact_id) as contact_count,
		       COUNT(*) as interaction_count, sentiment
		FROM interactions
		WHERE topic != '' AND topic IS NOT NULL
		GROUP BY topic
		ORDER BY interaction_count DESC
		LIMIT 20
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []TopicStats
	topicSentiments := make(map[string]map[string]int)

	for rows.Next() {
		var s TopicStats
		var sentiment sql.NullString

		err := rows.Scan(&s.Topic, &s.ContactCount, &s.InteractionCount, &sentiment)
		if err != nil {
			return nil, err
		}

		if sentiment.Valid {
			if topicSentiments[s.Topic] == nil {
				topicSentiments[s.Topic] = make(map[string]int)
			}
			topicSentiments[s.Topic][sentiment.String]++
		}

		s.AvgSentiment = getModalValue(topicSentiments[s.Topic])
		stats = append(stats, s)
	}

	return stats, nil
}

func (a *Analytics) GetToneStats(days int) ([]ToneStats, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	rows, err := a.db.conn.Query(`
		SELECT sentiment, COUNT(*) as count
		FROM interactions
		WHERE date >= ?
		GROUP BY sentiment
	`, cutoffDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []ToneStats
	var totalCount int

	for rows.Next() {
		var s ToneStats
		var sentiment sql.NullString

		err := rows.Scan(&sentiment, &s.Count)
		if err != nil {
			return nil, err
		}

		if sentiment.Valid {
			s.Sentiment = sentiment.String
		} else {
			s.Sentiment = "unknown"
		}

		totalCount += s.Count
		stats = append(stats, s)
	}

	// Calculate percentages
	for i := range stats {
		if totalCount > 0 {
			stats[i].Percentage = (float64(stats[i].Count) / float64(totalCount)) * 100
		}
		stats[i].Period = fmt.Sprintf("Last %d days", days)
	}

	// Sort by count descending
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	return stats, nil
}

func (a *Analytics) GetSummary() (*SummaryStats, error) {
	var summary SummaryStats

	// Total contacts
	a.db.conn.QueryRow("SELECT COUNT(*) FROM contacts").Scan(&summary.TotalContacts)

	// Total interactions
	a.db.conn.QueryRow("SELECT COUNT(*) FROM interactions").Scan(&summary.TotalInteractions)

	// Most frequent contact
	var mostFrequent sql.NullString
	a.db.conn.QueryRow(`
		SELECT name FROM contacts ORDER BY interaction_count DESC LIMIT 1
	`).Scan(&mostFrequent)
	if mostFrequent.Valid {
		summary.MostFrequentContact = mostFrequent.String
	}

	// Most common topic
	var mostTopic sql.NullString
	a.db.conn.QueryRow(`
		SELECT topic FROM interactions
		WHERE topic != '' AND topic IS NOT NULL
		GROUP BY topic ORDER BY COUNT(*) DESC LIMIT 1
	`).Scan(&mostTopic)
	if mostTopic.Valid {
		summary.MostCommonTopic = mostTopic.String
	}

	// Dominant tone
	var dominantTone sql.NullString
	a.db.conn.QueryRow(`
		SELECT sentiment FROM interactions
		WHERE sentiment != '' AND sentiment IS NOT NULL
		GROUP BY sentiment ORDER BY COUNT(*) DESC LIMIT 1
	`).Scan(&dominantTone)
	if dominantTone.Valid {
		summary.DominantTone = dominantTone.String
	}

	// Average topics per interaction
	var avgTopics sql.NullFloat64
	a.db.conn.QueryRow(`
		SELECT AVG(topic_count) FROM (
			SELECT COUNT(DISTINCT topic) as topic_count
			FROM interactions
			GROUP BY contact_id
		) t
	`).Scan(&avgTopics)
	if avgTopics.Valid {
		summary.AverageTopics = avgTopics.Float64
	}

	// New contacts this week
	weekAgo := time.Now().AddDate(0, 0, -7)
	a.db.conn.QueryRow(
		"SELECT COUNT(*) FROM contacts WHERE created_at >= ?",
		weekAgo,
	).Scan(&summary.NewContactsThisWeek)

	return &summary, nil
}

func (a *Analytics) GetCommunicationPatterns() ([]CommunicationPattern, error) {
	rows, err := a.db.conn.Query(`
		SELECT c.name, COUNT(i.id) as frequency, i.sentiment, i.topic, c.last_interaction
		FROM contacts c
		LEFT JOIN interactions i ON c.id = i.contact_id
		GROUP BY c.id
		ORDER BY frequency DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []CommunicationPattern
	for rows.Next() {
		var cp CommunicationPattern
		var sentiment sql.NullString
		var topic sql.NullString
		var lastInt sql.NullTime

		err := rows.Scan(&cp.Contact, &cp.Frequency, &sentiment, &topic, &lastInt)
		if err != nil {
			return nil, err
		}

		if sentiment.Valid {
			cp.AvgTone = sentiment.String
		} else {
			cp.AvgTone = "unknown"
		}

		if topic.Valid {
			cp.TopTopics = append(cp.TopTopics, topic.String)
		}

		if lastInt.Valid {
			cp.LastContact = lastInt.Time.Format("2006-01-02")
		}

		patterns = append(patterns, cp)
	}

	return patterns, nil
}

func (a *Analytics) GetTrendsByTimeframe(days int) (map[string]interface{}, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	tones, err := a.getTonesByTimeframe(cutoffDate)
	if err != nil {
		return nil, err
	}

	contacts, err := a.getActiveContactsByTimeframe(cutoffDate)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"timeframe":      fmt.Sprintf("Last %d days", days),
		"tone_breakdown": tones,
		"active_contacts": contacts,
	}, nil
}

func (a *Analytics) getTonesByTimeframe(since time.Time) (map[string]int, error) {
	rows, err := a.db.conn.Query(`
		SELECT sentiment, COUNT(*) as count
		FROM interactions
		WHERE date >= ?
		GROUP BY sentiment
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var sentiment sql.NullString
		var count int

		err := rows.Scan(&sentiment, &count)
		if err != nil {
			return nil, err
		}

		key := "unknown"
		if sentiment.Valid {
			key = sentiment.String
		}
		result[key] = count
	}

	return result, nil
}

func (a *Analytics) getActiveContactsByTimeframe(since time.Time) ([]map[string]interface{}, error) {
	rows, err := a.db.conn.Query(`
		SELECT c.name, COUNT(i.id) as count, MAX(i.date) as last_date
		FROM contacts c
		LEFT JOIN interactions i ON c.id = i.contact_id AND i.date >= ?
		WHERE i.id IS NOT NULL
		GROUP BY c.id
		ORDER BY count DESC
		LIMIT 10
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var name string
		var count int
		var lastDate sql.NullTime

		err := rows.Scan(&name, &count, &lastDate)
		if err != nil {
			return nil, err
		}

		item := map[string]interface{}{
			"contact": name,
			"interactions": count,
		}

		if lastDate.Valid {
			item["last_interaction"] = lastDate.Time.Format("2006-01-02")
		}

		result = append(result, item)
	}

	return result, nil
}

func getModalValue(m map[string]int) string {
	if len(m) == 0 {
		return "unknown"
	}

	maxKey := ""
	maxCount := 0

	for k, v := range m {
		if v > maxCount {
			maxCount = v
			maxKey = k
		}
	}

	return maxKey
}
