package cashshop

// MarkGiftNoteSentAndEmit implements MARK_GIFT_NOTE_SENT (task-240 Defect I):
// marks the locker asset identified by cashId as having had its gift-forward
// note sent, in whichever of accountId's compartments holds it. This is a
// SECOND, independent flag from GiftAcknowledged (see
// asset.Entity.GiftNoteSent's doc comment) -- it answers "has its note been
// sent?", set at note-send time, never at announce time. There is nothing
// client-facing to announce -- the channel already forwarded the note before
// sending this command -- so a partial failure across compartments is only
// logged, never surfaced, and does not roll back the compartments that
// already succeeded.
func (p *ProcessorImpl) MarkGiftNoteSentAndEmit(accountId uint32, cashId int64) error {
	ccms, err := p.cicP.GetByAccountId(accountId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to resolve compartments for account [%d] to mark gift note sent.", accountId)
		return err
	}

	var lastErr error
	for _, ccm := range ccms {
		if err := p.astP.MarkGiftNoteSent(ccm.Id(), cashId); err != nil {
			p.l.WithError(err).Errorf("Unable to mark gift note sent in compartment [%s] for account [%d].", ccm.Id(), accountId)
			lastErr = err
		}
	}
	return lastErr
}
