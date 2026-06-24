/**
 * Lightweight hash router — delegates rendering to the dashboard bootstrap.
 */
import { DEFAULT_PAGE, PAGE_META } from './routes.js';

let navigateImpl = null;

export function registerNavigator(fn) {
  navigateImpl = fn;
}

export function navigate(pageId, hostName, options) {
  if (navigateImpl) navigateImpl(pageId, hostName, options);
}

export function pageTitle(pageId) {
  return PAGE_META[pageId]?.title || PAGE_META[DEFAULT_PAGE].title;
}

export function pageCrumb(pageId) {
  return PAGE_META[pageId]?.crumb || '';
}
