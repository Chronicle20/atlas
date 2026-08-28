package cashshop

// AcknowledgeGiftsAndEmit implements ACKNOWLEDGE_GIFTS (task-240 Defect H):
// drains the "gift list presented" flag on every asset in accountId's
// compartments whose CashId appears in cashIds. There is nothing
// client-facing to announce -- the channel already fired LOAD_GIFT_SUCCESS
// before sending this command -- so a partial failure across compartments is
// only logged, never surfaced, and does not roll back the compartments that
// already succeeded.
func (p *ProcessorImpl) AcknowledgeGiftsAndEmit(accountId uint32, cashIds []int64) error {
	if len(cashIds) == 0 {
		return nil
	}

	ccms, err := p.cicP.GetByAccountId(accountId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to resolve compartments for account [%d] to acknowledge gifts.", accountId)
		return err
	}

	var lastErr error
	for _, ccm := range ccms {
		if err := p.astP.AcknowledgeGifts(ccm.Id(), cashIds); err != nil {
			p.l.WithError(err).Errorf("Unable to acknowledge gifts in compartment [%s] for account [%d].", ccm.Id(), accountId)
			lastErr = err
		}
	}
	return lastErr
}
