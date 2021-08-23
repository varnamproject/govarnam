#!/usr/bin/env ruby

require 'optparse'

'''
Compile a scheme file to VST
Requires libvarnam
TODO remove dependency on libvarnam
'''

def gem_available?(name)
	require name
rescue LoadError
   false
end

if not gem_available?('ffi')
  puts "Can't find gem - ffi. To install run '[sudo] gem install ffi'"
  exit(1)
end

$options = {}

$custom_lists = {}
$current_custom_list = []

# Starts a list context. Any tokens created inside will get added to this list
# It can have multiple list names and token will get added to all of these. One token
# can be in multiple lists
def list(*names, &block)
    if not $current_custom_list.empty?
        # This happens when user tries to nest list.
        # Nesting list is not allowed
        error "Can't create nested list"
        exit (1)
    end

    if names.empty?
        error "List should have a name"
        exit (1)
    end

    names.each do |name|
        if not name.is_a?(String) and not name.is_a?(Symbol)
            error "List name should be a string or symbols"
            exit (1)
        end

        $custom_lists[name] = [] if not $custom_lists.has_key?(name)
        $current_custom_list << $custom_lists[name]
    end

    yield if block_given?
ensure
    $current_custom_list = []
end

def push_to_current_custom_list(token)
    if token.nil?
        error "Can't add empty token"
        exit (1)
    end

    $current_custom_list.each do |l|
        l.push(token)
    end
end

# We handle method missing to return appropriate lists
def self.method_missing(name, *args, &block)
    return $custom_lists[name] if $custom_lists.has_key?(name)
    super
end

# this contains default symbols key overridden in the scheme file
# key will be the token type
$overridden_default_symbols = []

def _ensure_sanity_of_array(array)
  # Possibilities are
  #  [e1, e2]
  #  [e1, [e2,e3], e4]
  error "An empty array won't workout" if array.size == 0
  array.each do |element|
    if element.is_a?(Array)
      _ensure_sanity_of_array(element)
    else
      _ensure_type_safety(element)
    end
  end
end

def _ensure_sanity_of_element(element)
  if element.is_a?(Array)
    _ensure_sanity_of_array(element)
  else
    _ensure_type_safety(element)
    if element.is_a?(String) and element.length == 0
      error "Empty values are not allowed"
    end
  end
end

def _ensure_type_safety(element)
  valid_types = [Integer, String, Array]
  error "#{element.class} is not a valid type. Valid types are #{valid_types.to_s}" if not valid_types.include?(element.class)
end

def _ensure_sanity(hash)
  if not hash.is_a?(Hash)
    error "Expected a Hash, but got a #{hash.class}"
    exit 1
  end

  hash.each_pair do |key, value|
    _context.current_expression = "#{key} => #{value}"

    _ensure_sanity_of_element (key)
    _ensure_sanity_of_element (value)

    warn "#{value} has more than three elements. Additional elements specified will be ignored" if value.is_a?(Array) and value.size > 3

    _context.current_expression = nil
  end
end

def _extract_keys_values_and_persist(keys, values, token_type, match_type = Varnam::VARNAM_MATCH_EXACT, priority, accept_condition)
  keys.each do |key|
    if key.is_a?(Array)
      # This a possibility match
      key.flatten!
      _extract_keys_values_and_persist(key, values, token_type, Varnam::VARNAM_MATCH_POSSIBILITY, priority, accept_condition)
    else
      _persist_key_values(key, values, token_type, match_type, priority, accept_condition)
    end
  end
end

def _persist_key_values(pattern, values, token_type, match_type, priority, accept_condition)
  return if _context.errors > 0

  match = match_type == Varnam::VARNAM_MATCH_EXACT ? "EXACT" : "POSSIBILITY"

  if (values.is_a?(Array))
    values.flatten!
    value1 = values[0]
    value2 = values[1] if values.size >= 2
    value3 = values[2] if values.size >= 3
  else
    value1 = values
    value2 = ""
    value3 = ""
  end

  tag = _context.current_tag
  tag = "" if tag.nil?
  created = VarnamLibrary.varnam_create_token($varnam_handle.get_pointer(0), pattern, value1, value2, value3, tag, token_type, match_type, priority, accept_condition, 1)
  if created != 0
    error_message = VarnamLibrary.varnam_get_last_error($varnam_handle.get_pointer(0))
    error error_message
    return
  end

  _context.tokens[token_type] = [] if _context.tokens[token_type].nil?
  vtoken = VarnamToken.new(token_type, pattern, value1, value2, value3, tag, match_type, priority, accept_condition)
  _context.tokens[token_type].push(vtoken)
  push_to_current_custom_list vtoken
end

def flush_unsaved_changes
  saved = VarnamLibrary.varnam_flush_buffer($varnam_handle.get_pointer(0))
  if saved != 0
    error_message = VarnamLibrary.varnam_get_last_error($varnam_handle.get_pointer(0))
    error error_message
    return
  end
end

def infer_dead_consonants(infer)
  configured = VarnamLibrary.varnam_config($varnam_handle.get_pointer(0), Varnam::VARNAM_CONFIG_USE_DEAD_CONSONANTS, :int, infer ? 1 : 0)
  if configured != 0
    error_message = VarnamLibrary.varnam_get_last_error($varnam_handle.get_pointer(0))
    error error_message
    return
  end
end

def ignore_duplicates(ignore)
  configured = VarnamLibrary.varnam_config($varnam_handle.get_pointer(0), Varnam::VARNAM_CONFIG_IGNORE_DUPLICATE_TOKEN, :int, ignore ? 1 : 0)
  if configured != 0
    error_message = VarnamLibrary.varnam_get_last_error($varnam_handle.get_pointer(0))
    error error_message
    return
  end
end

def set_scheme_details()
	d = VarnamLibrary::SchemeDetails.new
	d[:langCode] = FFI::MemoryPointer.from_string($scheme_details[:langCode])
	d[:identifier] = FFI::MemoryPointer.from_string($scheme_details[:identifier])
	d[:displayName] = FFI::MemoryPointer.from_string($scheme_details[:displayName])
	d[:author] = FFI::MemoryPointer.from_string($scheme_details[:author])
	d[:compiledDate] = FFI::MemoryPointer.from_string(Time.now.to_s)
	if $scheme_details[:isStable].nil?
		d[:isStable] = 0
	else
		d[:isStable] = $scheme_details[:isStable]
	end

  done = VarnamLibrary.varnam_set_scheme_details($varnam_handle.get_pointer(0), d.pointer)
  if done != 0
    error_message = VarnamLibrary.varnam_get_last_error($varnam_handle.get_pointer(0))
    error error_message
    return
  end
end

$scheme_details = {}

def language_code(code)
	$scheme_details[:langCode] = code
end

def identifier(id)
	$scheme_details[:identifier] = id
end

def display_name(name)
	$scheme_details[:displayName] = name
end

def author(name)
	$scheme_details[:author] = name
end

def stable(value)
	$scheme_details[:isStable] = 0
	$scheme_details[:isStable] = 1 if value
end

def generate_cv
    all_vowels = get_vowels
    all_consonants = get_consonants

    all_consonants.each do |c|
        consonant_has_inherent_a_sound = c.pattern.end_with?('a') and not c.pattern[c.pattern.length - 2] == 'a'
        all_vowels.each do |v|
            next if v.value2.nil? or v.value2.length == 0

            if consonant_has_inherent_a_sound
                pattern = "#{c.pattern[0..c.pattern.length-2]}#{v.pattern}"
            else
                pattern = "#{c.pattern}#{v.pattern}"
            end

            values = ["#{c.value1}#{v.value2}"]
            if c.match_type == Varnam::VARNAM_MATCH_POSSIBILITY or v.match_type == Varnam::VARNAM_MATCH_POSSIBILITY
                match_type = Varnam::VARNAM_MATCH_POSSIBILITY
            else
                match_type = Varnam::VARNAM_MATCH_EXACT
            end

            accept_condition = nil
            if not v.accept_condition == Varnam::VARNAM_TOKEN_ACCEPT_ALL and not c.accept_condition == Varnam::VARNAM_TOKEN_ACCEPT_ALL
                accept_condition = v.accept_condition
            elsif not v.accept_condition == Varnam::VARNAM_TOKEN_ACCEPT_ALL
                accept_condition = v.accept_condition
            else
                accept_condition = c.accept_condition
            end

            priority = Varnam::VARNAM_TOKEN_PRIORITY_NORMAL
            if v.priority < c.priority
                priority = v.priority
            else
                priority = c.priority
            end


            _persist_key_values pattern, values, Varnam::VARNAM_TOKEN_CONSONANT_VOWEL, match_type, priority, accept_condition
        end
    end
end

def combine_array(array, is_pattern, replacements, current_item)
    if replacements.empty?
        error 'Replacements should be present when combining an array. This could be a bug within varnamc'
        exit (1)
    end

    result = []
    array.each do |a|
        if a.is_a?(Array)
            result.push(combine_array(a, is_pattern, replacements, current_item))
        else
            if is_pattern
                if current_item.match_type == Varnam::VARNAM_MATCH_POSSIBILITY
                    result.push([a.to_s.gsub("*", replacements[0])])
                else
                    result.push(a.to_s.gsub("*", replacements[0]))
                end
            else
                new_key = a.to_s.gsub("\*1", replacements[0])
                if replacements.length > 1 and not replacements[1].to_s.empty?
                    new_key = new_key.gsub("\*2", replacements[1])
                end
                if replacements.length > 2 and not replacements[2].to_s.empty?
                    new_key = new_key.gsub("\*3", replacements[2])
                end
                result.push (new_key)
            end
        end
    end

    return result
end

# Combines an array and a hash values
# This method also replaces the placeholder in hash
def combine(array, hash)
    _ensure_sanity(hash)
    if not array.is_a?(Array)
        error "Expected an array, but got a #{array.class}"
        exit 1
    end

    grouped = {}
    array.each do |item|
        hash.each_pair do |key, value|
            new_key = nil
            if key.is_a?(Array)
                new_key = combine_array(key, true, [item.pattern], item)
            else
                if item.match_type == Varnam::VARNAM_MATCH_POSSIBILITY
                    new_key = [[key.to_s.gsub("*", item.pattern)]]
                else
                    new_key = key.to_s.gsub("*", item.pattern)
                end
            end

            new_value = nil
            if value.is_a?(Array)
                new_value = combine_array(value, false, [item.value1, item.value2, item.value3], item)
            else
                new_value = value.to_s.gsub("\*1", item.value1)
                if not item.value2.nil? and not item.value2.to_s.empty?
                    new_value = new_value.gsub("\*2", item.value2)
                end
                if not item.value3.nil? and not item.value3.to_s.empty?
                    new_value = new_value.gsub("\*3", item.value3)
                end
            end

            if grouped[new_value].nil?
                grouped[new_value] = new_key
            else
                grouped[new_value].push(new_key)
            end
        end
    end

    # invert the hash
    result = {}
    grouped.each_pair do |key, value|
        result[value] = key
    end

    return result
end

def _create_token(hash, token_type, options = {})
  return if _context.errors > 0

  priority = _get_priority options
  accept_condition = _get_accept_condition options

  hash.each_pair do |key, value|
    if key.is_a?(Array)
      _extract_keys_values_and_persist(key, value, token_type, priority, accept_condition)
    else
      _persist_key_values(key, value, token_type, Varnam::VARNAM_MATCH_EXACT, priority, accept_condition)
    end
  end
end

def _validate_number(number, name)
    if not number.is_a?(Integer)
        error "#{name} should be a number"
        exit (1)
    end
end

def _get_priority(options)
    return Varnam::VARNAM_TOKEN_PRIORITY_NORMAL if options[:priority].nil? or options[:priority] == :normal
    return Varnam::VARNAM_TOKEN_PRIORITY_LOW if options[:priority] == :low
    return Varnam::VARNAM_TOKEN_PRIORITY_HIGH if options[:priority] == :high

    _validate_number options[:priority], "priority"

    return options[:priority]
end

def _get_accept_condition(options)
    return Varnam::VARNAM_TOKEN_ACCEPT_ALL if options[:accept_if].nil? or options[:accept_if] == :all
    return Varnam::VARNAM_TOKEN_ACCEPT_IF_STARTS_WITH if options[:accept_if] == :starts_with
    return Varnam::VARNAM_TOKEN_ACCEPT_IF_IN_BETWEEN if options[:accept_if] == :in_between
    return Varnam::VARNAM_TOKEN_ACCEPT_IF_ENDS_WITH if options[:accept_if] == :ends_with

    _validate_number options[:accept_if], "accept_if"
end

def vowels(options={}, hash)
  _ensure_sanity(hash)
  _create_token(hash, Varnam::VARNAM_TOKEN_VOWEL, options)
end

def consonants(options={}, hash)
  _ensure_sanity(hash)
  _create_token(hash, Varnam::VARNAM_TOKEN_CONSONANT, options)
end

def period(p)
	_create_token({"." => p}, Varnam::VARNAM_TOKEN_PERIOD, {})
end

def tag(name, &block)
   _context.current_tag = name
   block.call
   _context.current_tag = nil
end

def consonant_vowel_combinations(options={}, hash)
  _ensure_sanity(hash)
  _create_token(hash, Varnam::VARNAM_TOKEN_CONSONANT_VOWEL, options)
end

def anusvara(options={}, hash)
  _ensure_sanity(hash)
  _create_token(hash, Varnam::VARNAM_TOKEN_ANUSVARA, options)
end

def visarga(options={}, hash)
  _ensure_sanity(hash)
  _create_token(hash, Varnam::VARNAM_TOKEN_VISARGA, options)
end

def virama(options={}, hash)
  _ensure_sanity(hash)
  _create_token(hash, Varnam::VARNAM_TOKEN_VIRAMA, options)
end

def symbols(options={}, hash)
  _ensure_sanity(hash)
  _create_token(hash, Varnam::VARNAM_TOKEN_SYMBOL, options)
end

def numbers(options={}, hash)
  _ensure_sanity(hash)
  _create_token(hash, Varnam::VARNAM_TOKEN_NUMBER, options)
end

def others(options={}, hash)
  _ensure_sanity(hash)
  _create_token(hash, Varnam::VARNAM_TOKEN_OTHER, options)
end

def non_joiner(hash)
  _ensure_sanity(hash)
  _create_token(hash, Varnam::VARNAM_TOKEN_NON_JOINER);
  $overridden_default_symbols.push Varnam::VARNAM_TOKEN_NON_JOINER
end

def joiner(hash)
  _ensure_sanity(hash)
  _create_token(hash, Varnam::VARNAM_TOKEN_JOINER);
  $overridden_default_symbols.push Varnam::VARNAM_TOKEN_JOINER
end

def get_tokens(token_type, criteria = {})
  tokens = _context.tokens[token_type]
  if criteria.empty?
    return tokens
  elsif criteria[:exact]
    return tokens.find_all {|t| t.match_type == Varnam::VARNAM_MATCH_EXACT}
  else
    return tokens.find_all {|t| t.match_type == Varnam::VARNAM_MATCH_POSSIBILITY}
  end
end

def get_vowels(criteria = {})
  return get_tokens(Varnam::VARNAM_TOKEN_VOWEL, criteria)
end

def get_consonants(criteria = {})
  return get_tokens(Varnam::VARNAM_TOKEN_CONSONANT, criteria)
end

def get_consonant_vowel_combinations(criteria = {})
  return get_tokens(Varnam::VARNAM_TOKEN_CONSONANT_VOWEL, criteria)
end

def get_anusvara(criteria = {})
  return get_tokens(Varnam::VARNAM_TOKEN_ANUSVARA, criteria)
end

def get_visarga(criteria = {})
  return get_tokens(Varnam::VARNAM_TOKEN_VISARGA, criteria)
end

def get_symbols(criteria = {})
  return get_tokens(Varnam::VARNAM_TOKEN_SYMBOL, criteria)
end

def get_numbers(criteria = {})
  return get_tokens(Varnam::VARNAM_TOKEN_OTHER, criteria)
end

def get_chill()
  tokens = get_tokens(Varnam::VARNAM_TOKEN_CONSONANT, {:exact => true})
  return tokens.find_all {|t| t.tag == "chill"}
end

def get_virama
    tokens = get_tokens(Varnam::VARNAM_TOKEN_VIRAMA, {})
    if tokens.empty?
        error 'Virama is not set'
        exit (1)
    end
    return tokens[0]
end

def ffito_string(value)
  str = ""
  ptr = value.to_ptr
  if not ptr.null?
    str = ptr.read_string
    str.force_encoding('UTF-8')
  end
  return str
end

def get_dead_consonants(criteria = {})
  # dead consonants are infered by varnam. ruby wrapper don't know anything about it.
  token_type = Varnam::VARNAM_TOKEN_DEAD_CONSONANT
  token_ptr = FFI::MemoryPointer.new :pointer
  done = VarnamLibrary.varnam_get_all_tokens($varnam_handle.get_pointer(0), token_type, token_ptr);
  if done != 0
    error_message = VarnamLibrary.varnam_get_last_error($varnam_handle.get_pointer(0))
    error error_message
    return
  end

  size = VarnamLibrary.varray_length(token_ptr.get_pointer(0))
  i = 0
  _context.tokens[token_type] = [] if _context.tokens[token_type].nil?
  until i >= size
    tok = VarnamLibrary.varray_get(token_ptr.get_pointer(0), i)
    ptr = token_ptr.read_pointer
    item = VarnamLibrary::Token.new(tok)
    varnam_token = VarnamToken.new(item[:type],
                                   ffito_string(item[:pattern]), ffito_string(item[:value1]),
                                   ffito_string(item[:value2]), ffito_string(item[:value3]),
                                   ffito_string(item[:tag]), item[:match_type])
    _context.tokens[token_type].push(varnam_token)
    i += 1
  end
  return get_tokens(token_type, criteria)
end

def print_warnings_and_errors
  if _context.warnings > 0
    _context.warning_messages.each do |msg|
      puts msg
    end
  end

  if _context.errors > 0
    _context.error_messages.each do |msg|
      puts msg
    end
  end
end

# Sets default symbols if user has not set overridden in the scheme file
def set_default_symbols
  non_joiner "_" => "_"  if not $overridden_default_symbols.include?(Varnam::VARNAM_TOKEN_NON_JOINER)
  joiner "__" => "__"  if not $overridden_default_symbols.include?(Varnam::VARNAM_TOKEN_JOINER)
  symbols "-" => "-"
end

def _persist_stemrules(old_ending, new_ending)
  return if _context.errors > 0
  rc = VarnamLibrary.varnam_create_stemrule($varnam_handle.get_pointer(0), old_ending, new_ending)
  if rc != 0
    error_message = VarnamLibrary.varnam_get_last_error($varnam_handle.get_pointer(0))
    error error_message
  end
  return rc
end

def _create_stemrule(hash, options)
  return if _context.errors > 0
  hash.each_pair do |key,value|
    rc = _persist_stemrules(key, value)
    if rc != 0
      puts "could not create stemrule for " + key + ":" + value
    end
  end
end 

def stemrules(hash,options={})
 # _ensure_sanity(hash)
  _create_stemrule(hash, options) 
  puts VarnamLibrary.varnam_get_last_error($varnam_handle.get_pointer(0))
end

def exceptions_stem(hash, options={})
  hash.each_pair do |key,value|
    rc = VarnamLibrary.varnam_create_stem_exception($varnam_handle.get_pointer(0), key, value)
    if rc != 0
      puts "Could not create stemrule exception"
    end
  end
end

def initialize_varnam_handle()
  # Include libvarnam bindings in ruby
  require './varnamruby.rb'
  $varnam_handle = FFI::MemoryPointer.new :pointer

  init_error_msg = FFI::MemoryPointer.new(:pointer, 1)
  initialized = VarnamLibrary.varnam_init($vst_path, $varnam_handle, init_error_msg)

  if (initialized != 0)
    ptr = init_error_msg.read_pointer()
    msg = ptr.nil? ? "" : ptr.read_string
    puts "Varnam initialization failed #{msg}"
    exit(1)
  end

  if ($options[:debug])
    puts "Turning debug on"
    done = VarnamLibrary.varnam_enable_logging($varnam_handle.get_pointer(0), Varnam::VARNAM_LOG_DEBUG, DebugCallback);
    if done != 0
      error_message = VarnamLibrary.varnam_get_last_error($varnam_handle.get_pointer(0))
      puts "Unable to turn debugging on. #{error_message}"
      exit(1)
    end
  end
end

def add_weight_column()
  require "sqlite3"

  db = SQLite3::Database.new $vst_path
  db.execute "ALTER TABLE symbols ADD COLUMN weight INT"
end

def compile_scheme(scheme_path)
  file_name = File.basename(scheme_path)
  if file_name.include?(".")
    file_name = file_name.split(".")[0]
  end

  $vst_name = file_name + ".vst"
  $vst_path = File.join(Dir.pwd, $vst_name)

  if File.exists?($vst_path)
    File.delete($vst_path)
  end

  initialize_varnam_handle()

  puts "Compiling #{scheme_path}"
  puts "Building #{$vst_name}"

  at_exit {
    print_warnings_and_errors if _context.errors > 0
    puts "Completed with '#{_context.warnings}' warning(s) and '#{_context.errors}' error(s)"
  }

  load scheme_path
  set_default_symbols
  flush_unsaved_changes
  set_scheme_details

  add_weight_column

  if _context.errors > 0
    returncode = 1
  else
    returncode = 0
  end

  exit(returncode)
end

def find_libvarnam
  return $options[:library] if not $options[:library].nil?
  # Trying to find out libvarnam in the predefined locations if
  # absolute path to the library is not specified
  libvarnam_search_paths = ['.', File.dirname(File.expand_path(__FILE__)), '/usr/local/lib', '/usr/local/lib/i386-linux-gnu', '/usr/local/lib/x86_64-linux-gnu', '/usr/lib/i386-linux-gnu', '/usr/lib/x86_64-linux-gnu', '/usr/lib']
  libvarnam_names = ['libvarnam.so', "libvarnam.so.#{$libvarnam_major_version}", 'libvarnam.dylib', 'varnam.dll']
  libvarnam_search_paths.each do |path|
    libvarnam_names.each do |fname|
      fullpath = File.join(path, fname)
      if File.exists?(fullpath)
        return fullpath
      end
    end
  end
  return nil
end

optparse = OptionParser.new do |opts|
  opts.banner = "Usage: compile-schema options"

  # ability to provide varnam library name
  $options[:library] = nil
  opts.on('-l', '--library FILE', 'Sets the varnam library') do |file|
    if not File.exist?(file)
      puts "Can't find #{file}"
      exit 1
    end
    $options[:library] = file
  end

  if $options[:library].nil?
    $options[:library] = find_libvarnam
    if $options[:library].nil?
      puts "Can't find varnam shared library. Try specifying the full path using -l option"
      puts optparse
    else
      puts "Using #{$options[:library]}" if $options[:verbose]
    end
  end

  $options[:debug] = false
  opts.on('-z', '--debug', 'Enable debugging') do
    $options[:debug] = true
  end

  opts.on('-s', '-s path_to_scheme_file_path', 'Path to scheme file') do |path|
    if File.exists? (path)
      compile_scheme(path)
    else
      puts "File doesn't exist"
    end
  end
end

optparse.parse!
