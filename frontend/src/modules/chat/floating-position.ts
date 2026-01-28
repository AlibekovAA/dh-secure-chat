export type AnchorRect = {
  left: number;
  right: number;
  top: number;
  bottom: number;
  width: number;
  height: number;
};

type BoundsRect = {
  left: number;
  top: number;
  right: number;
  bottom: number;
};

export function computeFloatingPosition(params: {
  anchorRect: AnchorRect;
  isOwn: boolean;
  popupWidth: number;
  popupHeight: number;
  padding: number;
  offset: number;
  boundsRect: BoundsRect;
}) {
  const {
    anchorRect,
    isOwn,
    popupWidth,
    popupHeight,
    padding,
    offset,
    boundsRect,
  } = params;

  const preferredBelow = anchorRect.bottom + offset;
  const preferredAbove = anchorRect.top - popupHeight - offset;
  const preferredLeft = isOwn ? anchorRect.right - popupWidth : anchorRect.left;

  const minX = boundsRect.left + padding;
  const maxX = boundsRect.right - popupWidth - padding;
  const minY = boundsRect.top + padding;
  const maxY = boundsRect.bottom - popupHeight - padding;

  const belowFits = preferredBelow + popupHeight <= boundsRect.bottom - padding;
  const aboveFits = preferredAbove >= boundsRect.top + padding;
  const top = belowFits
    ? preferredBelow
    : aboveFits
      ? preferredAbove
      : preferredBelow;

  return {
    x: Math.min(Math.max(preferredLeft, minX), maxX),
    y: Math.min(Math.max(top, minY), maxY),
  };
}
