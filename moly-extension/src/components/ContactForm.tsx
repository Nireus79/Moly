/**
 * Contact Form Component
 * Add or edit contacts
 */

import React, { useState, useEffect } from 'react';
import { useContactStore } from '@/stores/contactStore';
import type { Contact } from '@/types';
import './contactForm.css';

interface ContactFormProps {
  contact?: Contact;
  onClose?: () => void;
  onSave?: (contact: Contact) => void;
}

export const ContactForm: React.FC<ContactFormProps> = ({ contact, onClose, onSave }) => {
  const { addContact, updateContact } = useContactStore();
  const [formData, setFormData] = useState({
    name: '',
    platform: 'other',
    notes: '',
  });
  const [error, setError] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (contact) {
      setFormData({
        name: contact.name,
        platform: contact.platform || 'other',
        notes: contact.notes || '',
      });
    }
  }, [contact]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
    setError('');
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (!formData.name.trim()) {
      setError('Contact name is required');
      return;
    }

    setIsSaving(true);

    try {
      if (contact) {
        await updateContact(contact.id, {
          name: formData.name,
          platform: formData.platform,
          notes: formData.notes,
        });
      } else {
        await addContact({
          name: formData.name,
          platform: formData.platform,
          notes: formData.notes,
        });
      }

      onSave?.(formData as any);
      onClose?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save contact');
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="contact-form">
      <div className="form-header">
        <h2>{contact ? 'Edit Contact' : 'New Contact'}</h2>
        <button className="close-btn" onClick={onClose} aria-label="Close">
          X
        </button>
      </div>

      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label htmlFor="name" className="form-label">
            Contact Name *
          </label>
          <input
            id="name"
            type="text"
            name="name"
            value={formData.name}
            onChange={handleChange}
            placeholder="e.g., Alex from Tinder"
            className="form-input"
            autoFocus
            disabled={isSaving}
          />
        </div>

        <div className="form-group">
          <label htmlFor="platform" className="form-label">
            Platform
          </label>
          <select
            id="platform"
            name="platform"
            value={formData.platform}
            onChange={handleChange}
            className="form-input"
            disabled={isSaving}
          >
            <option value="tinder">Tinder</option>
            <option value="bumble">Bumble</option>
            <option value="hinge">Hinge</option>
            <option value="facebook">Facebook</option>
            <option value="instagram">Instagram</option>
            <option value="linkedin">LinkedIn</option>
            <option value="discord">Discord</option>
            <option value="slack">Slack</option>
            <option value="twitter">Twitter/X</option>
            <option value="telegram">Telegram</option>
            <option value="whatsapp">WhatsApp</option>
            <option value="other">Other</option>
          </select>
        </div>

        <div className="form-group">
          <label htmlFor="notes" className="form-label">
            Notes
          </label>
          <textarea
            id="notes"
            name="notes"
            value={formData.notes}
            onChange={handleChange}
            placeholder="Add any notes or context about this contact..."
            className="form-textarea"
            rows={3}
            disabled={isSaving}
          />
        </div>

        {error && <div className="form-error">{error}</div>}

        <div className="form-actions">
          <button type="button" onClick={onClose} className="btn btn-secondary" disabled={isSaving}>
            Cancel
          </button>
          <button type="submit" className="btn btn-primary" disabled={isSaving}>
            {isSaving ? 'Saving...' : contact ? 'Update' : 'Add Contact'}
          </button>
        </div>
      </form>
    </div>
  );
};

export default ContactForm;
