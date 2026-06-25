create type invitation_status as enum ('pending', 'accepted', 'declined', 'ignored', 'revoked');

create table if not exists invitations
(
    inviter        text                            not null,
    invitee        text                            not null,
    organization   uuid,
    created_at     timestamp without time zone     not null default CURRENT_TIMESTAMP,
    referral_code  varchar(8)                      not null,
    status         invitation_status               not null default 'pending'
);

alter table if exists invitations
    add constraint referral_code_unique unique (referral_code);

alter table if exists invitations
    add constraint inviter_fk foreign key (inviter) references users (email);

create index if not exists referral_code_hash_idx
    on invitations using hash (referral_code);

create index if not exists inviter_invitee_unique_idx
    on invitations (inviter, invitee);
