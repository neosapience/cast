package client

import "encoding/json"

type SubscriptionResponse struct {
	Plan    string  `json:"plan"`
	Credits Credits `json:"credits"`
	Limits  Limits  `json:"limits"`
}

type Credits struct {
	PlanCredits int64 `json:"plan_credits"`
	UsedCredits int64 `json:"used_credits"`
}

type Limits struct {
	ConcurrencyLimit int `json:"concurrency_limit"`
}

func (c *Client) GetMySubscription() (*SubscriptionResponse, error) {
	data, err := c.get("/v1/users/me/subscription")
	if err != nil {
		return nil, err
	}

	var sub SubscriptionResponse
	if err := json.Unmarshal(data, &sub); err != nil {
		return nil, err
	}

	return &sub, nil
}
