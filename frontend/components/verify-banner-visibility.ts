/**
 * The banner must wait for /auth/me to SETTLE (roleResolved), not merely for
 * hydration (ready): `ready` flips true at hydration while `emailVerified` is
 * still its default `false`, which made the banner show for verified users on
 * every hard load (and stick if getMe errored).
 */
export function shouldShowVerifyBanner(s: {
  ready: boolean;
  isAuthed: boolean;
  roleResolved: boolean;
  emailVerified: boolean;
}): boolean {
  return s.ready && s.isAuthed && s.roleResolved && !s.emailVerified;
}
