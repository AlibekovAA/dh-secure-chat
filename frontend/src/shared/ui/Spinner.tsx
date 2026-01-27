type SpinnerSize = 'xs' | 'sm' | 'md' | 'lg';

type SpinnerProps = {
  size?: SpinnerSize;
  borderColorClass?: string;
  className?: string;
};

const sizeClassMap: Record<SpinnerSize, string> = {
  xs: 'w-3 h-3',
  sm: 'w-5 h-5',
  md: 'w-6 h-6',
  lg: 'w-8 h-8',
};

export function Spinner({
  size = 'md',
  borderColorClass = 'border-emerald-400',
  className = '',
}: SpinnerProps) {
  const sizeClass = sizeClassMap[size];

  return (
    <div
      className={`border-2 border-t-transparent rounded-full animate-spin ${sizeClass} ${borderColorClass} ${className}`}
    />
  );
}
