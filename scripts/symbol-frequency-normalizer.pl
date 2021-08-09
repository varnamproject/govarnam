# Reweigh the words
# Sample: word="",f=8671269 to word="",f=200
# Source: command line argument
# Output to terminal

# Biggest value = # of lines.
# Divide this by 240 and round up (255-14 to avoid 0-15 values)
# Divide all other values (lines left in the list) by that number and round down.
# All values should now be between 15 and 254.


# Open original file
use utf8;
open FILE, $ARGV[0] or die $!;
my $count=0;

# Count the # of lines
while (<FILE>) {
    $count++;
}

# Calculate the divider to ensure results between 0 and 100
my $divider = int( $count / 100) + 1 ;

sub is_integer { $_[0] =~ /^[+-]?\d+$/ }
# Re-open the source file and update the weight
open FILE, "<:encoding(utf8)", $ARGV[0] or die $!;

# remove ’, “, ।, —, ‘, ·, −, °, ”, ॥
while (my $line = <FILE>) {
    $count--;

    # Replace the weight if its a word line,
    # otherwise print without actions
    if ($line =~ /\s/) {
        my $weighed = int( $count / $divider) + 0;
        my ($name) = $line =~ m/(.*)\s/;
        if (length($name) > 1 && !is_integer($name)) {
            $line =~ s/(\d*[.])?\d+/$weighed/g;
            utf8::encode($line);
            print $line;
        }
    }
}

close FILE;
