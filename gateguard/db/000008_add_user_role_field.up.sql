alter table invitations
    add column if not exists invitee_role text;

update invitations
set invitee_role = 'common';

alter table invitations
    alter column invitee_role set not null;
